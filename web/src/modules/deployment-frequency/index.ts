import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";
import type { AppState } from "~/store";
import { InsightMetricsKind, InsightDataPoint } from "../insight";
import * as InsightAPI from "~/api/insight";
import { LoadingStatus } from "~/types/module";
import {
  InsightResolution,
  InsightResultType,
} from "pipecd/web/model/insight_pb";
import { determineTimeRange } from "~/modules/insight";
import dayjs from "dayjs";

const MODULE_NAME = "deploymentFrequency";

export interface DeploymentFrequencyState {
  status: LoadingStatus;
  data: InsightDataPoint.AsObject[];
  status24h: LoadingStatus;
  data24h: InsightDataPoint.AsObject[];
}

const initialState: DeploymentFrequencyState = {
  status: "idle",
  data: [],
  status24h: "idle",
  data24h: [],
};

export const fetchDeploymentFrequency = createAsyncThunk<
  InsightDataPoint.AsObject[],
  void,
  { state: AppState }
>(`${MODULE_NAME}/fetch`, async (_, thunkAPI) => {
  const state = thunkAPI.getState();

  const labels = new Array<[string, string]>();
  if (state.insight.labels) {
    for (const label of state.insight.labels) {
      const pair = label.split(":");
      if (pair.length === 2) labels.push([pair[0], pair[1]]);
    }
  }

  const [rangeFrom, rangeTo] = determineTimeRange(
    state.insight.range,
    state.insight.resolution
  );

  const data = await InsightAPI.getInsightData({
    metricsKind: InsightMetricsKind.DEPLOYMENT_FREQUENCY,
    rangeFrom: rangeFrom,
    rangeTo: rangeTo,
    resolution: state.insight.resolution,
    applicationId: state.insight.applicationId,
    labelsMap: labels,
  });

  if (data.type == InsightResultType.MATRIX) {
    return data.matrixList[0].dataPointsList;
  } else {
    return [];
  }
});

export const fetchDeployment24h = createAsyncThunk<
  InsightDataPoint.AsObject[],
  void,
  { state: AppState }
>(`${MODULE_NAME}/fetch24h`, async () => {
  const rangeTo = dayjs.utc().endOf("day").valueOf();
  const rangeFrom = dayjs.utc().startOf("day").valueOf();

  const data = await InsightAPI.getInsightData({
    metricsKind: InsightMetricsKind.DEPLOYMENT_FREQUENCY,
    rangeFrom,
    rangeTo,
    resolution: InsightResolution.DAILY,
    applicationId: "",
    labelsMap: [],
  });
  if (data.type == InsightResultType.MATRIX) {
    return data.matrixList[0].dataPointsList;
  } else {
    return [];
  }
});

export const deploymentFrequencySlice = createSlice({
  name: MODULE_NAME,
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchDeploymentFrequency.pending, (state) => {
        state.status = "loading";
      })
      .addCase(fetchDeploymentFrequency.rejected, (state) => {
        state.status = "failed";
      })
      .addCase(fetchDeploymentFrequency.fulfilled, (state, action) => {
        state.status = "succeeded";
        state.data = action.payload;
      })
      .addCase(fetchDeployment24h.pending, (state) => {
        state.status24h = "loading";
      })
      .addCase(fetchDeployment24h.rejected, (state) => {
        state.status24h = "failed";
      })
      .addCase(fetchDeployment24h.fulfilled, (state, action) => {
        state.status24h = "succeeded";
        state.data24h = action.payload;
      });
  },
});
