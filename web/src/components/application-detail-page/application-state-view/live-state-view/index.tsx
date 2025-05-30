import { Box } from "@mui/material";
import { FC, useEffect, useMemo, useState } from "react";

import { ResourceState } from "~~/model/application_live_state_pb";
import DeployTargetTabBar from "./deploy-target-tab-bar";
import GraphView from "./graph-view";

type Props = {
  resources: ResourceState.AsObject[];
};

export const LiveStateView: FC<Props> = ({ resources: allResources }) => {
  const [deployTargetTabSelected, setDeployTargetTabSelected] = useState("");

  const resourcesByDeployTarget = useMemo(() => {
    return allResources.reduce((all, resource) => {
      const deployTarget = resource.deployTarget;
      if (!deployTarget) return all;

      if (!all[deployTarget]) {
        all[deployTarget] = [];
      }
      all[deployTarget].push(resource);
      return all;
    }, {} as Record<string, ResourceState.AsObject[]>);
  }, [allResources]);

  const deployTargetList = useMemo(
    () => Object.keys(resourcesByDeployTarget).sort(),
    [resourcesByDeployTarget]
  );

  useEffect(() => {
    if (deployTargetTabSelected == "" && deployTargetList.length > 0) {
      setDeployTargetTabSelected(deployTargetList[0]);
    }
  }, [deployTargetList, deployTargetTabSelected]);

  const resources = useMemo(() => {
    if (!deployTargetTabSelected) return [];
    return resourcesByDeployTarget[deployTargetTabSelected];
  }, [deployTargetTabSelected, resourcesByDeployTarget]);

  return (
    <Box
      sx={{
        display: "flex",
        flex: 1,
        justifyContent: "center",
        overflow: "hidden",
        position: "relative",
      }}
    >
      <Box sx={{ position: "absolute", left: 0, top: 0, zIndex: 100 }}>
        <DeployTargetTabBar
          tabs={deployTargetList}
          selectedTab={deployTargetTabSelected}
          onSelectTab={setDeployTargetTabSelected}
        />
      </Box>

      {deployTargetTabSelected && (
        <GraphView key={deployTargetTabSelected} resources={resources} />
      )}
    </Box>
  );
};
