// Copyright 2024 The PipeCD Authors.
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

package ecs

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/pipe-cd/pipecd/pkg/app/piped/executor"
	"github.com/pipe-cd/pipecd/pkg/app/piped/platformprovider"
	provider "github.com/pipe-cd/pipecd/pkg/app/piped/platformprovider/ecs"
	"github.com/pipe-cd/pipecd/pkg/config"
	"github.com/pipe-cd/pipecd/pkg/model"
)

type rollbackExecutor struct {
	executor.Input
}

func (e *rollbackExecutor) Execute(sig executor.StopSignal) model.StageStatus {
	var (
		ctx            = sig.Context()
		originalStatus = e.Stage.Status
		status         model.StageStatus
	)

	switch model.Stage(e.Stage.Name) {
	case model.StageRollback:
		status = e.ensureRollback(ctx)
	default:
		e.LogPersister.Errorf("Unsupported stage %s for ECS application", e.Stage.Name)
		return model.StageStatus_STAGE_FAILURE
	}

	return executor.DetermineStageStatus(sig.Signal(), originalStatus, status)
}

func (e *rollbackExecutor) ensureRollback(ctx context.Context) model.StageStatus {
	// Not rollback in case this is the first deployment.
	if e.Deployment.RunningCommitHash == "" {
		e.LogPersister.Errorf("Unable to determine the last deployed commit to rollback. It seems this is the first deployment.")
		return model.StageStatus_STAGE_FAILURE
	}

	runningDS, err := e.RunningDSP.GetReadOnly(ctx, e.LogPersister)
	if err != nil {
		e.LogPersister.Errorf("Failed to prepare running deploy source data (%v)", err)
		return model.StageStatus_STAGE_FAILURE
	}

	appCfg := runningDS.ApplicationConfig.ECSApplicationSpec
	if appCfg == nil {
		e.LogPersister.Errorf("Malformed application configuration: missing ECSApplicationSpec")
		return model.StageStatus_STAGE_FAILURE
	}

	platformProviderName, platformProviderCfg, found := findPlatformProvider(&e.Input)
	if !found {
		return model.StageStatus_STAGE_FAILURE
	}

	taskDefinition, ok := loadTaskDefinition(&e.Input, appCfg.Input.TaskDefinitionFile, runningDS)
	if !ok {
		return model.StageStatus_STAGE_FAILURE
	}
	serviceDefinition, ok := loadServiceDefinition(&e.Input, appCfg.Input.ServiceDefinitionFile, runningDS)
	if !ok {
		return model.StageStatus_STAGE_FAILURE
	}

	primary, canary, ok := loadTargetGroups(&e.Input, appCfg, runningDS)
	if !ok {
		return model.StageStatus_STAGE_FAILURE
	}

	if !rollback(ctx, &e.Input, platformProviderName, platformProviderCfg, taskDefinition, serviceDefinition, primary, canary) {
		return model.StageStatus_STAGE_FAILURE
	}

	return model.StageStatus_STAGE_SUCCESS
}

func rollback(ctx context.Context, in *executor.Input, platformProviderName string, platformProviderCfg *config.PlatformProviderECSConfig, taskDefinition types.TaskDefinition, serviceDefinition types.Service, primaryTargetGroup *types.LoadBalancer, canaryTargetGroup *types.LoadBalancer) bool {
	in.LogPersister.Infof("Start rollback the ECS service and task family: %s and %s to original stage", *serviceDefinition.ServiceName, *taskDefinition.Family)
	client, err := provider.DefaultRegistry().Client(platformProviderName, platformProviderCfg, in.Logger)
	if err != nil {
		in.LogPersister.Errorf("Unable to create ECS client for the provider %s: %v", platformProviderName, err)
		return false
	}

	// Re-register TaskDef to get TaskDefArn.
	// Consider using DescribeServices and get services[0].taskSets[0].taskDefinition (taskDefinition of PRIMARY taskSet)
	// then store it in metadata store and use for rollback instead.
	td, err := client.RegisterTaskDefinition(ctx, taskDefinition)
	if err != nil {
		in.LogPersister.Errorf("Failed to register new revision of ECS task definition %s: %v", *taskDefinition.Family, err)
		return false
	}

	// Rollback ECS service configuration to previous state including commit-hash of the tag.
	service, err := applyServiceDefinition(ctx, client, serviceDefinition)
	if err != nil {
		in.LogPersister.Errorf("Unable to rollback ECS service %s configuration to previous stage: %v", *serviceDefinition.ServiceName, err)
		return false
	}

	// Get current PRIMARY/ACTIVE task set.
	prevTaskSets, err := client.GetServiceTaskSets(ctx, *service)
	// Ignore error in case it's not found error, the prevTaskSets doesn't exist for newly created Service.
	if err != nil && !errors.Is(err, platformprovider.ErrNotFound) {
		in.LogPersister.Errorf("Failed to determine current ECS PRIMARY/ACTIVE taskSet of service %s for rollback: %v", *serviceDefinition.ServiceName, err)
		return false
	}

	// On rolling back, the scale of desired tasks will be set to 100 (same as the original state).
	taskSet, err := client.CreateTaskSet(ctx, *service, *td, primaryTargetGroup, 100)
	if err != nil {
		in.LogPersister.Errorf("Failed to create ECS task set %s: %v", *serviceDefinition.ServiceName, err)
		return false
	}

	// Make new taskSet as PRIMARY task set, so that it will handle production service.
	if _, err = client.UpdateServicePrimaryTaskSet(ctx, *service, *taskSet); err != nil {
		in.LogPersister.Errorf("Failed to update PRIMARY ECS taskSet for service %s: %v", *serviceDefinition.ServiceName, err)
		return false
	}

	// Reset routing in case of rolling back progressive pipeline.
	if primaryTargetGroup != nil {
		if !rollbackELB(ctx, in, client, primaryTargetGroup, canaryTargetGroup) {
			return false
		}
	}

	// Delete previous ACTIVE taskSets
	in.LogPersister.Infof("Start deleting previous ACTIVE taskSets")
	for _, ts := range prevTaskSets {
		in.LogPersister.Infof("Deleting previous ACTIVE taskSet %s", *ts.TaskSetArn)
		if err := client.DeleteTaskSet(ctx, *ts); err != nil {
			in.LogPersister.Errorf("Failed to remove previous ACTIVE taskSet %s: %v", *ts.TaskSetArn, err)
			return false
		}
	}

	in.LogPersister.Infof("Rolled back the ECS service %s and task definition %s configuration to original stage", *serviceDefinition.ServiceName, *taskDefinition.Family)
	return true
}

func rollbackELB(ctx context.Context, in *executor.Input, client provider.Client, primaryTargetGroup *types.LoadBalancer, canaryTargetGroup *types.LoadBalancer) bool {
	var canaryTargetGroupArn string
	if canaryTargetGroup == nil {
		// Get the touched canary target group from a TRAFFIC_ROUTING stage.
		// canaryTargetGroup does not exist in the previous commit when the attempted deployment newly introduced it.
		var ok bool
		canaryTargetGroupArn, ok = in.MetadataStore.Shared().Get(canaryTargetGroupArnKey)
		if !ok {
			in.LogPersister.Infof("Skip rolling back ELB listeners because it seems the deployment failed before updating them")
			return true
		}
		in.LogPersister.Infof("Successfully got canary target group ARN from metadata store, although it was not included in the last successful commit: %s", canaryTargetGroupArn)
	} else {
		// When canaryTargetGroup exists in the previous commit, simply use it.
		canaryTargetGroupArn = *canaryTargetGroup.TargetGroupArn
	}

	routingTrafficCfg := provider.RoutingTrafficConfig{
		{
			TargetGroupArn: *primaryTargetGroup.TargetGroupArn,
			Weight:         100,
		},
		{
			TargetGroupArn: canaryTargetGroupArn,
			Weight:         0,
		},
	}

	currListenerArns, err := client.GetListenerArns(ctx, *primaryTargetGroup)
	if err != nil {
		in.LogPersister.Errorf("Failed to get current active listeners: %v", err)
		return false
	}

	modifiedRules, err := client.ModifyListeners(ctx, currListenerArns, routingTrafficCfg)
	if err != nil {
		in.LogPersister.Errorf("Failed to routing traffic to PRIMARY/CANARY variants: %v", err)

		if len(modifiedRules) > 0 {
			logModifiedRules(in.LogPersister, modifiedRules)
		}
		return false
	}

	logModifiedRules(in.LogPersister, modifiedRules)

	in.LogPersister.Infof("Successfully rolled back ELB listeners of target groups %s (PRIMARY) and %s (CANARY)", *primaryTargetGroup.TargetGroupArn, canaryTargetGroupArn)
	return true
}
