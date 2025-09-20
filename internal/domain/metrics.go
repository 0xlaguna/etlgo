package domain

import (
	"time"
)

// represents calculated business metrics
type BusinessMetrics struct {
	Date        time.Time `json:"date"`
	Channel     string    `json:"channel"`
	CampaignID  string    `json:"campaign_id"`
	UTMCampaign string    `json:"utm_campaign"`
	UTMSource   string    `json:"utm_source"`
	UTMMedium   string    `json:"utm_medium"`

	// Raw metrics
	Clicks        int     `json:"clicks"`
	Impressions   int     `json:"impressions"`
	Cost          float64 `json:"cost"`
	Leads         int     `json:"leads"`
	Opportunities int     `json:"opportunities"`
	ClosedWon     int     `json:"closed_won"`
	Revenue       float64 `json:"revenue"`

	// Calculated metrics
	CPC          float64 `json:"cpc"`
	CPA          float64 `json:"cpa"`
	CVRLeadToOpp float64 `json:"cvr_lead_to_opp"`
	CVROppToWon  float64 `json:"cvr_opp_to_won"`
	ROAS         float64 `json:"roas"`

	// Metadata
	CalculatedAt time.Time `json:"calculated_at"`
}

// represents filters for querying metrics
type MetricsFilter struct {
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	Channel     string     `json:"channel,omitempty"`
	CampaignID  string     `json:"campaign_id,omitempty"`
	UTMCampaign string     `json:"utm_campaign,omitempty"`
	UTMSource   string     `json:"utm_source,omitempty"`
	UTMMedium   string     `json:"utm_medium,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// represents the API response for metrics queries
type MetricsResponse struct {
	Data    []BusinessMetrics `json:"data"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}

// represents data structure for export functionality
type ExportData struct {
	Date          string  `json:"date"`
	Channel       string  `json:"channel"`
	CampaignID    string  `json:"campaign_id"`
	Clicks        int     `json:"clicks"`
	Impressions   int     `json:"impressions"`
	Cost          float64 `json:"cost"`
	Leads         int     `json:"leads"`
	Opportunities int     `json:"opportunities"`
	ClosedWon     int     `json:"closed_won"`
	Revenue       float64 `json:"revenue"`
	CPC           float64 `json:"cpc"`
	CPA           float64 `json:"cpa"`
	CVRLeadToOpp  float64 `json:"cvr_lead_to_opp"`
	CVROppToWon   float64 `json:"cvr_opp_to_won"`
	ROAS          float64 `json:"roas"`
}
