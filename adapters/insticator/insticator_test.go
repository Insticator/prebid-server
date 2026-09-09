package insticator

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderInsticator, config.Adapter{
		Endpoint:         "https://insticator.example.com/v1/pbs",
		ExtraAdapterInfo: `{"app_endpoint": "https://insticator-app.example.com/v1/pbs"}`,
	},
		config.Server{ExternalUrl: "https://insticator.example.com/v1/pbs", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "insticatortest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name         string
		mType        openrtb2.MarkupType
		bidExt       json.RawMessage
		imps         []openrtb2.Imp
		expectedType openrtb_ext.BidType
	}{
		{
			name:         "banner markup maps to banner",
			mType:        openrtb2.MarkupBanner,
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "video markup maps to video",
			mType:        openrtb2.MarkupVideo,
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			name:         "audio markup maps to audio",
			mType:        openrtb2.MarkupAudio,
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "markup wins over a conflicting ext media type",
			mType:        openrtb2.MarkupAudio,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"banner"}}`),
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "absent markup ignores an ext media type with no impression to infer from",
			mType:        0,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"audio"}}`),
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "the impression wins over a conflicting ext media type",
			mType:        0,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"video"}}`),
			imps:         []openrtb2.Imp{{ID: "imp-1", Audio: &openrtb2.Audio{MIMEs: []string{"audio/mp4"}}}},
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "unknown markup infers from the impression rather than the ext",
			mType:        openrtb2.MarkupNative,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"banner"}}`),
			imps:         []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}}}},
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			name:         "absent markup with no ext falls back to banner",
			mType:        0,
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "absent markup with an unparsable ext falls back to banner",
			mType:        0,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"not-a-media-type"}}`),
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "absent markup with malformed ext falls back to banner",
			mType:        0,
			bidExt:       json.RawMessage(`{`),
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "absent markup infers audio from an audio-only impression",
			mType:        0,
			imps:         []openrtb2.Imp{{ID: "imp-1", Audio: &openrtb2.Audio{MIMEs: []string{"audio/mp4"}}}},
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "absent markup infers video from a video-only impression",
			mType:        0,
			imps:         []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}}}},
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			name:         "absent markup stays banner when the impression is multi format",
			mType:        0,
			imps:         []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}, Audio: &openrtb2.Audio{MIMEs: []string{"audio/mp4"}}}},
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "absent markup stays banner when no impression matches the bid",
			mType:        0,
			imps:         []openrtb2.Imp{{ID: "other-imp", Audio: &openrtb2.Audio{MIMEs: []string{"audio/mp4"}}}},
			expectedType: openrtb_ext.BidTypeBanner,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bid := &openrtb2.Bid{ImpID: "imp-1", MType: test.mType, Ext: test.bidExt}
			assert.Equal(t, test.expectedType, getMediaTypeForBid(bid, test.imps))
		})
	}
}

func TestGetBidMetaMediaType(t *testing.T) {
	tests := []struct {
		name         string
		bidType      openrtb_ext.BidType
		expectedType string
	}{
		{name: "audio", bidType: openrtb_ext.BidTypeAudio, expectedType: "audio"},
		{name: "video", bidType: openrtb_ext.BidTypeVideo, expectedType: "video"},
		{name: "banner", bidType: openrtb_ext.BidTypeBanner, expectedType: "banner"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bid := &openrtb2.Bid{ADomain: []string{"insticator.com"}, Cat: []string{"IAB1-1"}}
			meta := getBidMeta(bid, test.bidType, "insticator")

			assert.Equal(t, test.expectedType, meta.MediaType)
			assert.Equal(t, "insticator", meta.Seat)
			assert.Equal(t, []string{"insticator.com"}, meta.AdvertiserDomains)
			assert.Equal(t, "IAB1-1", meta.PrimaryCategoryID)
		})
	}
}

func TestGetBidVideoOnlyForVideo(t *testing.T) {
	bid := &openrtb2.Bid{Dur: 30, Cat: []string{"IAB1-1"}}

	assert.Nil(t, getBidVideo(bid, openrtb_ext.BidTypeAudio), "audio bids carry no bid video")
	assert.Nil(t, getBidVideo(bid, openrtb_ext.BidTypeBanner), "banner bids carry no bid video")

	video := getBidVideo(bid, openrtb_ext.BidTypeVideo)
	assert.NotNil(t, video)
	assert.Equal(t, 30, video.Duration)
	assert.Equal(t, "IAB1-1", video.PrimaryCategory)
}
