package insticator

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// type adapter struct {
// 	endpoint string
// }

type Ext struct {
	Insticator impInsticatorExt `json:"insticator"`
}

type impInsticatorExt struct {
	Prod   string `json:"prod"`
	Zoneid string `json:"zoneid,omitempty"`
}

type adapter struct {
	endpoint string
}

type reqExt struct {
	Insticator *reqInsticatorExt `json:"insticator,omitempty"`
}

type reqInsticatorExt struct {
	Caller []InsticatorCaller `json:"caller,omitempty"`
}

type InsticatorCaller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// CALLER Info used to track Prebid Server
// as one of the hops in the request to exchange
var CALLER = InsticatorCaller{"Prebid-Server", "n/a"}

type bidExt struct {
	Insticator bidInsticatorExt `json:"insticator,omitempty"`
}

type bidInsticatorExt struct {
	MediaType string `json:"mediaType,omitempty"`
}

// Builder builds a new insticatorance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForBid(bid *openrtb2.Bid) openrtb_ext.BidType {
	// Log the bid.MType
	log.Printf("bid.MType: %v", bid.MType)
	// Log the entire bid structure
	bidJson, err := json.MarshalIndent(bid, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal bid: %v", err)
	} else {
		log.Printf("Bid: %s", bidJson)
	}

	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	default:
		return openrtb_ext.BidTypeBanner
	}
}

func getBidType(ext bidExt) openrtb_ext.BidType {
	if ext.Insticator.MediaType == "video" {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// requestJSON, err := json.Marshal(request)
	// if err != nil {
	// 	return nil, []error{err}
	// }

	// requestData := &adapters.RequestData{
	// 	Method: "POST",
	// 	Uri:    a.endpoint,
	// 	Body:   requestJSON,
	// }

	// return []*adapters.RequestData{requestData}, nil

	var errs []error
	var adapterRequests []*adapters.RequestData
	var groupedImps = make(map[string][]openrtb2.Imp)

	// Construct request extension common to all imps
	// NOTE: not blocking adapter requests on errors
	// since request extension is optional.
	reqExt, err := makeReqExt(request)
	if err != nil {
		errs = append(errs, err)
	}
	request.Ext = reqExt

	// We only support SRA for requests containing same prod and
	// zoneID, therefore group all imps accordingly and create a http
	// request for each such group
	for i := 0; i < len(request.Imp); i++ {
		if impCopy, err := makeImps(request.Imp[i]); err == nil {
			var impExt Ext

			// Skip over imps whose extensions cannot be read since
			// we cannot glean Prod or ZoneID which are required to
			// group together. However let's not block request creation.
			if err := json.Unmarshal(impCopy.Ext, &impExt); err == nil {
				impKey := impExt.Insticator.Prod + impExt.Insticator.Zoneid
				groupedImps[impKey] = append(groupedImps[impKey], impCopy)
			} else {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}

	for _, impList := range groupedImps {
		if adapterReq, err := a.makeRequest(*request, impList); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errs = append(errs, err)
		}
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request openrtb2.BidRequest, impList []openrtb2.Imp) (*adapters.RequestData, error) {
	request.Imp = impList

	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType := getMediaTypeForBid(bid)
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func makeImps(imp openrtb2.Imp) (openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: fmt.Sprintf("Imp ID %s must have at least one of [Banner, Video] defined", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var insticatorExt openrtb_ext.ExtImp33across
	if err := json.Unmarshal(bidderExt.Bidder, &insticatorExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var impExt Ext
	impExt.Insticator.Prod = insticatorExt.ProductId

	impExt.Insticator.Zoneid = insticatorExt.SiteId

	if len(insticatorExt.ZoneId) > 0 {
		impExt.Insticator.Zoneid = insticatorExt.ZoneId
	}

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON

	// Validate Video if it exists
	if imp.Video != nil {
		videoCopy, err := validateVideoParams(imp.Video, impExt.Insticator.Prod)

		imp.Video = videoCopy

		if err != nil {
			return openrtb2.Imp{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	return imp, nil
}

func validateVideoParams(video *openrtb2.Video, prod string) (*openrtb2.Video, error) {
	videoCopy := *video
	if (videoCopy.W == nil || *videoCopy.W == 0) ||
		(videoCopy.H == nil || *videoCopy.H == 0) ||
		videoCopy.Protocols == nil ||
		videoCopy.MIMEs == nil ||
		videoCopy.PlaybackMethod == nil {

		return nil, &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, protocols, mimes, playbackmethod",
		}
	}

	if videoCopy.Placement == 0 {
		videoCopy.Placement = 2
	}

	if prod == "insticatorream" {
		videoCopy.Placement = 1

		if videoCopy.StartDelay == nil {
			videoCopy.StartDelay = adcom1.StartDelay.Ptr(0)
		}
	}

	return &videoCopy, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt reqExt

	if len(request.Ext) > 0 {
		if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
			return nil, err
		}
	}

	if reqExt.Insticator == nil {
		reqExt.Insticator = &reqInsticatorExt{}
	}

	if reqExt.Insticator.Caller == nil {
		reqExt.Insticator.Caller = make([]InsticatorCaller, 0)
	}

	reqExt.Insticator.Caller = append(reqExt.Insticator.Caller, CALLER)

	return json.Marshal(reqExt)
}
