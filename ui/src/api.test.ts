import axios, { AxiosRequestConfig, AxiosResponse } from "axios";
import { setCSRFToken } from "./api";

function successfulResponse(config: AxiosRequestConfig): AxiosResponse {
  return {
    config,
    data: {},
    headers: {},
    status: 200,
    statusText: "OK",
  };
}

afterEach(() => {
  setCSRFToken();
});

it("adds the CSRF token to state-changing requests", async () => {
  setCSRFToken("csrf-token");
  let requestConfig: AxiosRequestConfig | undefined;

  await axios.post(
    "/api/queues/default:pause",
    {},
    {
      adapter: async (config) => {
        requestConfig = config;
        return successfulResponse(config);
      },
    }
  );

  expect(requestConfig?.headers?.["X-CSRF-Token"]).toBe("csrf-token");
});
