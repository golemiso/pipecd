import { FC } from "react";
import { InsightDataPoint, InsightResolution } from "~/modules/insight";
import { ChartBase } from "../chart-base";
import { blue as chartColor } from "@mui/material/colors";

export interface DeploymentFrequencyChartProps {
  resolution: InsightResolution;
  data: { name: string; points: InsightDataPoint.AsObject[] }[];
}

export const DeploymentFrequencyChart: FC<DeploymentFrequencyChartProps> = ({
  resolution,
  data,
}) => {
  return (
    <ChartBase
      title="Deployment Frequency"
      resolution={resolution}
      data={data}
      xName=""
      yName="Number of Deployments"
      yMax={undefined}
      lineColor={chartColor[500]}
      areaColor={chartColor[200]}
    />
  );
};
