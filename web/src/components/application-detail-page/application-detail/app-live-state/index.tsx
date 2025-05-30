import { Box, Typography } from "@mui/material";
import Skeleton from "@mui/material/Skeleton";
import { FC, memo } from "react";
import { APPLICATION_HEALTH_STATUS_TEXT } from "~/constants/health-status-text";
import { UI_TEXT_NOT_AVAILABLE_TEXT } from "~/constants/ui-text";
import { useAppSelector } from "~/hooks/redux";
import {
  ApplicationLiveState,
  selectById,
  selectLoadingById,
} from "~/modules/applications-live-state";
import { ApplicationHealthStatusIcon } from "../health-status-icon";

export interface AppLiveStateProps {
  applicationId: string;
}

export const AppLiveState: FC<AppLiveStateProps> = memo(function AppLiveState({
  applicationId,
}) {
  const [liveState, liveStateLoading] = useAppSelector<
    [ApplicationLiveState | undefined, boolean]
  >((state) => [
    selectById(state.applicationLiveState, applicationId),
    selectLoadingById(state.applicationLiveState, applicationId),
  ]);

  if (liveStateLoading && liveState === undefined) {
    return <Skeleton height={32} width={100} />;
  }

  return (
    <Box
      sx={{
        display: "flex",
        alignItems: "center",
      }}
    >
      {liveState ? (
        <ApplicationHealthStatusIcon health={liveState.healthStatus} />
      ) : null}
      <Typography
        variant="h6"
        sx={{
          ml: 0.5,
        }}
      >
        {liveState
          ? APPLICATION_HEALTH_STATUS_TEXT[liveState.healthStatus]
          : UI_TEXT_NOT_AVAILABLE_TEXT}
      </Typography>
    </Box>
  );
});
