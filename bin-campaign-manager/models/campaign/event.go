package campaign

// list of campaign event types
const (
	EventTypeCampaignCreated string = "campaign_created" // the campaign created
	EventTypeCampaignUpdated string = "campaign_updated" // the campaign updated
	EventTypeCampaignDeleted string = "campaign_deleted" // the campaign deleted

	EventTypeCampaignStatusRun      string = "campaign_status_run"      // the campaign updated
	EventTypeCampaignStatusStopping string = "campaign_status_stopping" // the campaign updated
	EventTypeCampaignStatusStop     string = "campaign_status_stop"     // the campaign updated
)
