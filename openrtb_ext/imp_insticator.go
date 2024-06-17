package openrtb_ext

// ExtImpInsticator defines the contract for bidrequest.imp[i].ext.prebid.bidder.insticator
type ExtImpInsticator struct {
	ZoneId    string `json:"zoneId,omitempty"`
	AdUnitId  string `json:"adUnitId,omitempty"`
	ProductId string `json:"productId,omitempty"`
}
