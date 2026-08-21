package models

type Prerequisites struct {
	AwsCliInstalled      bool     `json:"awsCliInstalled"`
	AwsCliVersion        string   `json:"awsCliVersion"`
	SessionPluginFound   bool     `json:"sessionPluginFound"`
	SessionPluginVersion string   `json:"sessionPluginVersion"`
	AllOk                bool     `json:"allOk"`
	Message              []string `json:"message"`
}
