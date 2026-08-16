import parseFlagsUnderWindow from "./parseFlags";

describe("parseFlagsUnderWindow", () => {
  it("parses values rendered by the Go server", () => {
    window.FLAG_ROOT_PATH = "/monitoring";
    window.FLAG_PROMETHEUS_SERVER_ADDRESS = "http://prometheus:9090";
    window.FLAG_READ_ONLY = "true";

    parseFlagsUnderWindow();

    expect(window.ROOT_PATH).toBe("/monitoring");
    expect(window.PROMETHEUS_SERVER_ADDRESS).toBe("http://prometheus:9090");
    expect(window.READ_ONLY).toBe(true);
  });

  it("uses safe defaults when Go template actions have not been rendered", () => {
    window.FLAG_ROOT_PATH = "";
    window.FLAG_PROMETHEUS_SERVER_ADDRESS = "/[[.PrometheusAddr]]";
    window.FLAG_READ_ONLY = "/[[.ReadOnly]]";

    parseFlagsUnderWindow();

    expect(window.ROOT_PATH).toBe("");
    expect(window.PROMETHEUS_SERVER_ADDRESS).toBe("");
    expect(window.READ_ONLY).toBe(false);
  });
});
