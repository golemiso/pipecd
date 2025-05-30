// Copyright 2025 The PipeCD Authors.
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

package main

import (
	"context"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

const (
	stageWait string = "WAIT"
)

type plugin struct{}

// BuildPipelineSyncStages implements sdk.StagePlugin.
func (p *plugin) BuildPipelineSyncStages(ctx context.Context, _ sdk.ConfigNone, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	stages := make([]sdk.PipelineStage, 0, len(input.Request.Stages))
	for _, rs := range input.Request.Stages {
		stage := sdk.PipelineStage{
			Index:              rs.Index,
			Name:               rs.Name,
			Rollback:           false,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		}
		stages = append(stages, stage)
	}

	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: stages,
	}, nil
}

// ExecuteStage implements sdk.StagePlugin.
func (p *plugin) ExecuteStage(ctx context.Context, _ sdk.ConfigNone, _ sdk.DeployTargetsNone, input *sdk.ExecuteStageInput[struct{}]) (*sdk.ExecuteStageResponse, error) {
	status := p.executeWait(ctx, input)
	return &sdk.ExecuteStageResponse{
		Status: status,
	}, nil
}

// FetchDefinedStages implements sdk.StagePlugin.
func (p *plugin) FetchDefinedStages() []string {
	return []string{stageWait}
}
