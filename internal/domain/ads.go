package domain

import "time"

type AdPerformance struct {
	Date        string  `json:"date"`
	CampaignID  string  `json:"campaign_id"`
	Channel     string  `json:"channel"`
	Clicks      int     `json:"clicks"`
	Impressions int     `json:"impressions"`
	Cost        float64 `json:"cost"`
	UTMCampaign string  `json:"utm_campaign"`
	UTMSource   string  `json:"utm_source"`
	UTMMedium   string  `json:"utm_medium"`
}

type AdData struct {
	External struct {
		Ads struct {
			Performance []AdPerformance `json:"performance"`
		} `json:"ads"`
	} `json:"external"`
}

type ProcessedAdData struct {
	Date        time.Time `json:"date"`
	CampaignID  string    `json:"campaign_id"`
	Channel     string    `json:"channel"`
	Clicks      int       `json:"clicks"`
	Impressions int       `json:"impressions"`
	Cost        float64   `json:"cost"`
	UTMCampaign string    `json:"utm_campaign"`
	UTMSource   string    `json:"utm_source"`
	UTMMedium   string    `json:"utm_medium"`
	ProcessedAt time.Time `json:"processed_at"`
}

// UTM combination for data correlation
type UTMKey struct {
	Campaign string
	Source   string
	Medium   string
}

// String returns a string representation of UTMKey for use as map key
func (u UTMKey) String() string {
	return u.Campaign + "|" + u.Source + "|" + u.Medium
}
