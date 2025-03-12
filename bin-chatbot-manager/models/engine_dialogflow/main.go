package engine_dialogflow

type EngineDialogflow struct {
	// common
	CredentialBase64 string `json:"credential_base64,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	Region           Region `json:"region,omitempty"`

	// CX only
	AgentID string `json:"agent_id,omitempty"`
}

type Region string

const (
	RegionNone   Region = ""
	RegionGlobal Region = "global" // ES available

	RegionUSCentral1 Region = "us-central1"
	RegionUSEast1    Region = "us-east1"
	RegionUSWest1    Region = "us-west1"
	RegionUSMulti    Region = "us-multi"

	RegionNorthAmericaNorthEast Region = "northamerica-northeast1"

	RegsionEuropeWest1 Region = "europe-west1" // ES available
	RegionEuropeWest2  Region = "europe-west2" // ES available
	RegionEuropeWest3  Region = "europe-west3"
	RegionEuropeWest4  Region = "europe-west4"
	RegionEuropeWest6  Region = "europe-west6"

	RegionAustraliaSouthEast Region = "australia-southeast1" // ES available

	RegionAsiaNorthEast1 Region = "asia-northeast1" // ES available
	RegionAsiaSouth1     Region = "asia-south1"
	RegionAsiaSouthEast1 Region = "asia-southeast1"
	RegionAsiaSouthEast2 Region = "asia-southeast2"
)
