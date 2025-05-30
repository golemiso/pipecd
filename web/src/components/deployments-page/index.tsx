import {
  Box,
  Button,
  CircularProgress,
  Divider,
  List,
  Toolbar,
  Typography,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import FilterIcon from "@mui/icons-material/FilterList";
import RefreshIcon from "@mui/icons-material/Refresh";
import dayjs from "dayjs";
import { FC, useCallback, useEffect, useRef, useState } from "react";
import { useInView } from "react-intersection-observer";
import { useNavigate } from "react-router-dom";
import { PAGE_PATH_DEPLOYMENTS } from "~/constants/path";
import {
  UI_TEXT_FILTER,
  UI_TEXT_HIDE_FILTER,
  UI_TEXT_REFRESH,
  UI_TEXT_MORE,
} from "~/constants/ui-text";
import {
  useAppDispatch,
  useAppSelector,
  useShallowEqualSelector,
} from "~/hooks/redux";
import { fetchApplications } from "~/modules/applications";
import {
  Deployment,
  DeploymentFilterOptions,
  fetchDeployments,
  fetchMoreDeployments,
  selectById as selectDeploymentById,
  selectIds as selectDeploymentIds,
} from "~/modules/deployments";
import { SpinnerIcon } from "~/styles/button";
import {
  stringifySearchParams,
  useSearchParams,
  arrayFormat,
} from "~/utils/search-params";
import { DeploymentFilter } from "./deployment-filter";
import { DeploymentItem } from "./deployment-item";

const sortComp = (a: string | number, b: string | number): number => {
  return dayjs(b).valueOf() - dayjs(a).valueOf();
};

function filterUndefined<TValue>(value: TValue | undefined): value is TValue {
  return value !== undefined;
}

const useGroupedDeployments = (): Record<string, Deployment.AsObject[]> => {
  const deployments = useShallowEqualSelector<Deployment.AsObject[]>((state) =>
    selectDeploymentIds(state.deployments)
      .map((id) => selectDeploymentById(state.deployments, id))
      .filter(filterUndefined)
  );

  const result: Record<string, Deployment.AsObject[]> = {};

  deployments.forEach((deployment) => {
    const dateStr = dayjs(deployment.createdAt * 1000).format("YYYY/MM/DD");
    if (!result[dateStr]) {
      result[dateStr] = [];
    }
    result[dateStr].push(deployment);
  });

  return result;
};

export const DeploymentIndexPage: FC = () => {
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const listRef = useRef(null);
  const status = useAppSelector((state) => state.deployments.status);
  const hasMore = useAppSelector((state) => state.deployments.hasMore);
  const groupedDeployments = useGroupedDeployments();
  const filterOptions = useSearchParams();
  const [openFilter, setOpenFilter] = useState(true);
  const [ref, inView] = useInView({
    rootMargin: "400px",
    root: listRef.current,
  });

  const isLoading = status === "loading";

  useEffect(() => {
    dispatch(fetchApplications());
  }, [dispatch]);

  useEffect(() => {
    dispatch(fetchDeployments(filterOptions));
  }, [dispatch, filterOptions]);

  useEffect(() => {
    if (inView && hasMore && isLoading === false) {
      dispatch(fetchMoreDeployments(filterOptions));
    }
  }, [dispatch, inView, hasMore, isLoading, filterOptions]);

  // filter handlers
  const handleFilterChange = useCallback(
    (options: DeploymentFilterOptions) => {
      navigate(
        `${PAGE_PATH_DEPLOYMENTS}?${stringifySearchParams(
          { ...options },
          { arrayFormat: arrayFormat }
        )}`,
        { replace: true }
      );
    },
    [navigate]
  );
  const handleFilterClear = useCallback(() => {
    navigate(PAGE_PATH_DEPLOYMENTS, { replace: true });
  }, [navigate]);

  const handleRefreshClick = useCallback(() => {
    dispatch(fetchDeployments(filterOptions));
  }, [dispatch, filterOptions]);

  const handleMoreClick = useCallback(() => {
    dispatch(fetchMoreDeployments(filterOptions));
  }, [dispatch, filterOptions]);

  const dates = Object.keys(groupedDeployments).sort(sortComp);

  return (
    <Box
      sx={{
        display: "flex",
        overflow: "hidden",
        flex: 1,
        flexDirection: "column",
      }}
    >
      <Toolbar variant="dense">
        <Box
          sx={{
            flexGrow: 1,
          }}
        />
        <Button
          color="primary"
          startIcon={<RefreshIcon />}
          onClick={handleRefreshClick}
          disabled={isLoading}
        >
          {UI_TEXT_REFRESH}
          {isLoading && <SpinnerIcon />}
        </Button>
        <Button
          color="primary"
          startIcon={openFilter ? <CloseIcon /> : <FilterIcon />}
          onClick={() => setOpenFilter(!openFilter)}
        >
          {openFilter ? UI_TEXT_HIDE_FILTER : UI_TEXT_FILTER}
        </Button>
      </Toolbar>
      <Divider />
      <Box
        sx={{
          display: "flex",
          overflow: "hidden",
          flex: 1,
        }}
      >
        <Box
          component={"ol"}
          sx={(theme) => ({
            listStyle: "none",
            padding: theme.spacing(3),
            paddingTop: 0,
            margin: 0,
            flex: 1,
            overflowY: "scroll",
          })}
          ref={listRef}
        >
          {dates.length === 0 &&
            (isLoading ? (
              <Box
                sx={{
                  display: "flex",
                  justifyContent: "center",
                  mt: 3,
                }}
              >
                <CircularProgress />
              </Box>
            ) : (
              <Box
                sx={{
                  display: "flex",
                  justifyContent: "center",
                  mt: 3,
                }}
              >
                <Typography>No deployments</Typography>
              </Box>
            ))}
          {dates.map((date) => (
            <li key={date}>
              <Typography variant="subtitle1" sx={{ mt: 2, mb: 2 }}>
                {date}
              </Typography>
              <List>
                {groupedDeployments[date]
                  .sort((a, b) => sortComp(a.createdAt, b.createdAt))
                  .map((deployment) => (
                    <DeploymentItem
                      id={deployment.id}
                      key={`deployment-item-${deployment.id}`}
                    />
                  ))}
              </List>
            </li>
          ))}
          {status === "succeeded" && <div ref={ref} />}
          {!hasMore && (
            <Button
              color="primary"
              variant="outlined"
              size="large"
              fullWidth
              onClick={handleMoreClick}
              disabled={isLoading}
            >
              {UI_TEXT_MORE}
              {isLoading && <SpinnerIcon />}
            </Button>
          )}
          {/* TODO: Show how many days have been read */}
        </Box>
        {openFilter && (
          <DeploymentFilter
            options={filterOptions}
            onChange={handleFilterChange}
            onClear={handleFilterClear}
          />
        )}
      </Box>
    </Box>
  );
};
