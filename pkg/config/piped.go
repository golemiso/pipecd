// Copyright 2020 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"fmt"

	"github.com/kapetaniosci/pipe/pkg/model"
)

var DefaultKubernetesCloudProvider = PipedCloudProvider{
	Name:             "kubernetes-default",
	Type:             model.CloudProviderKubernetes,
	KubernetesConfig: &CloudProviderKubernetesConfig{},
}

// PipedSpec contains configurable data used to while running Piped.
type PipedSpec struct {
	// How often to check whether an application should be synced.
	SyncInterval Duration `json:"syncInterval"`
	// Git configuration needed for git commands.
	Git PipedGit `json:"git"`
	// List of git repositories this piped will handle.
	Repositories      []PipedRepository    `json:"repositories"`
	CloudProviders    []PipedCloudProvider `json:"cloudProviders"`
	AnalysisProviders []AnalysisProvider   `json:"analysisProviders"`
}

// Validate validates configured data of all fields.
func (s *PipedSpec) Validate() error {
	return nil
}

// EnableDefaultKubernetesCloudProvider adds the default kubernetes cloud provider if it was not specified.
func (s *PipedSpec) EnableDefaultKubernetesCloudProvider() {
	for _, cp := range s.CloudProviders {
		if cp.Name == DefaultKubernetesCloudProvider.Name {
			return
		}
	}
	s.CloudProviders = append(s.CloudProviders, DefaultKubernetesCloudProvider)
}

// GetRepositoryMap returns a map of repositories where key is repo id.
func (s *PipedSpec) GetRepositoryMap() map[string]PipedRepository {
	m := make(map[string]PipedRepository, len(s.Repositories))
	for _, repo := range s.Repositories {
		m[repo.RepoID] = repo
	}
	return m
}

// GetRepository finds a repository with the given ID from the configured list.
func (s *PipedSpec) GetRepository(id string) (PipedRepository, bool) {
	for _, repo := range s.Repositories {
		if repo.RepoID == id {
			return repo, true
		}
	}
	return PipedRepository{}, false
}

// GetProvider finds and returns an Analysis Provider config whose name is the given string.
func (s *PipedSpec) GetProvider(name string) (AnalysisProvider, bool) {
	for _, p := range s.AnalysisProviders {
		if p.Name == name {
			return p, true
		}
	}
	return AnalysisProvider{}, false
}

type PipedGit struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	// Where to write ssh config file.
	// Default is "/etc/ssh/ssh_config".
	SSHConfigFilePath string `json:"sshConfigFilePath"`
	// The host name.
	// e.g. github.com, gitlab.com
	Host string `json:"host"`
	// The hostname or IP address of the remote git server.
	// e.g. github.com, gitlab.com
	HostName string `json:"hostName"`
	// The path to the private ssh key file.
	// This will be used to clone the source code of the git repositories.
	SSHKeyFile string `json:"sshKeyFile"`
	// The path to the GitHub/GitLab access token file.
	// This will be used to authenticate while creating pull request...
	AccessTokenFile string `json:"accessTokenFile"`
}

func (g PipedGit) ShouldConfigureSSHConfig() bool {
	if g.SSHConfigFilePath != "" {
		return true
	}
	if g.Host != "" {
		return true
	}
	if g.HostName != "" {
		return true
	}
	if g.SSHKeyFile != "" {
		return true
	}
	return false
}

type PipedRepository struct {
	// Unique identifier for this repository.
	// This must be unique in the piped scope.
	RepoID string `json:"repoId"`
	// Remote address of the repository.
	// e.g. git@github.com:org/repo1.git
	Remote string `json:"remote"`
	// The branch should be tracked.
	Branch string `json:"branch"`
}

type PipedCloudProvider struct {
	Name string
	Type model.CloudProviderType

	KubernetesConfig *CloudProviderKubernetesConfig
	TerraformConfig  *CloudProviderTerraformConfig
	CloudRunConfig   *CloudProviderCloudRunConfig
	LambdaConfig     *CloudProviderLambdaConfig
}
type genericPipedCloudProvider struct {
	Name   string                  `json:"name"`
	Type   model.CloudProviderType `json:"type"`
	Config json.RawMessage         `json:"config"`
}

func (p *PipedCloudProvider) UnmarshalJSON(data []byte) error {
	var err error
	gp := genericPipedCloudProvider{}
	if err = json.Unmarshal(data, &gp); err != nil {
		return err
	}
	p.Name = gp.Name
	p.Type = gp.Type

	switch p.Type {
	case model.CloudProviderKubernetes:
		p.KubernetesConfig = &CloudProviderKubernetesConfig{}
		if len(gp.Config) > 0 {
			err = json.Unmarshal(gp.Config, p.KubernetesConfig)
		}
	case model.CloudProviderTerraform:
		p.TerraformConfig = &CloudProviderTerraformConfig{}
		if len(gp.Config) > 0 {
			err = json.Unmarshal(gp.Config, p.TerraformConfig)
		}
	case model.CloudProviderCloudRun:
		p.CloudRunConfig = &CloudProviderCloudRunConfig{}
		if len(gp.Config) > 0 {
			err = json.Unmarshal(gp.Config, p.CloudRunConfig)
		}
	case model.CloudProviderLambda:
		p.LambdaConfig = &CloudProviderLambdaConfig{}
		if len(gp.Config) > 0 {
			err = json.Unmarshal(gp.Config, p.LambdaConfig)
		}
	default:
		err = fmt.Errorf("unsupported cloud provider type: %s", p.Name)
	}
	return err
}

type CloudProviderKubernetesConfig struct {
	//AllowNamespaces []string `json:"allowNamespaces"`
	MasterURL      string `json:"masterURL"`
	KubeConfigPath string `json:"kubeConfigPath"`
}

type CloudProviderTerraformConfig struct {
	GCP *CloudProviderTerraformGCP `json:"gcp"`
	AWS *CloudProviderTerraformAWS `json:"aws"`
}

type CloudProviderTerraformGCP struct {
	Project         string `json:"project"`
	Region          string `json:"region"`
	CredentialsFile string `json:"credentialsFile"`
}

type CloudProviderTerraformAWS struct {
	Region string `json:"region"`
}

type CloudProviderCloudRunConfig struct {
	Project         string `json:"project"`
	Region          string `json:"region"`
	Platform        string `json:"platform"`
	CredentialsFile string `json:"credentialsFile"`
}

type CloudProviderLambdaConfig struct {
	Region string `json:"region"`
}

type AnalysisProvider struct {
	Name        string                       `json:"name"`
	Prometheus  *AnalysisProviderPrometheus  `json:"prometheus"`
	Datadog     *AnalysisProviderDatadog     `json:"datadog"`
	Stackdriver *AnalysisProviderStackdriver `json:"stackdriver"`
}

type AnalysisProviderPrometheus struct {
	Address string `json:"address"`
	// The path to the username file.
	UsernameFile string `json:"usernameFile"`
	// The path to the password file.
	PasswordFile string `json:"passwordFile"`
}

type AnalysisProviderDatadog struct {
	Address string `json:"address"`
	// The path to the api key file.
	APIKeyFile string `json:"apiKeyFile"`
	// The path to the application key file.
	ApplicationKeyFile string `json:"applicationKeyFile"`
}

type AnalysisProviderStackdriver struct {
	// The path to the service account file.
	ServiceAccountFile string `json:"serviceAccountFile"`
}
