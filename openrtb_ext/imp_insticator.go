package openrtb_ext

// ExtInsticator defines the contract for bidrequest.imp[i].ext.prebid.bidder.insticator
type ExtInsticator struct {
	SiteId    string `json:"siteId"`
	ZoneId    string `json:"zoneId,omitempty"`
	ProductId string `json:"productId,omitempty"`
}
