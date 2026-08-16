import React from "react";
import { createTheme, ThemeProvider } from "@material-ui/core/styles";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import AuthGate from "./AuthGate";
import {
  getAuthSession,
  login,
  logout,
  setCSRFToken,
  setUnauthorizedHandler,
} from "../api";

vi.mock("../api", () => ({
  getAuthSession: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  setCSRFToken: vi.fn(),
  setUnauthorizedHandler: vi.fn(),
}));

const mockedGetAuthSession = vi.mocked(getAuthSession);
const mockedLogin = vi.mocked(login);
const mockedLogout = vi.mocked(logout);
const mockedSetCSRFToken = vi.mocked(setCSRFToken);
const mockedSetUnauthorizedHandler = vi.mocked(setUnauthorizedHandler);

function renderAuthGate() {
  return render(
    <ThemeProvider theme={createTheme()}>
      <AuthGate>
        {(session) => (
          <div>
            <p>Dashboard for {session.username || "anonymous"}</p>
            {session.authEnabled && (
              <button onClick={session.signOut}>Test sign out</button>
            )}
          </div>
        )}
      </AuthGate>
    </ThemeProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockedLogout.mockResolvedValue();
});

it("signs in, stores the CSRF token, and signs out", async () => {
  mockedGetAuthSession.mockResolvedValue({
    enabled: true,
    authenticated: false,
  });
  mockedLogin.mockResolvedValue({
    enabled: true,
    authenticated: true,
    username: "operator",
    csrfToken: "csrf-token",
  });

  renderAuthGate();

  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeInTheDocument();
  await userEvent.type(screen.getByLabelText("Username"), "operator");
  await userEvent.type(screen.getByLabelText("Password"), "secret");
  await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

  expect(await screen.findByText("Dashboard for operator")).toBeInTheDocument();
  expect(mockedLogin).toHaveBeenCalledWith("operator", "secret");
  expect(mockedSetCSRFToken).toHaveBeenCalledWith("csrf-token");

  await userEvent.click(screen.getByRole("button", { name: "Test sign out" }));
  await waitFor(() => expect(mockedLogout).toHaveBeenCalledTimes(1));
  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeInTheDocument();
});

it("renders the application immediately when authentication is disabled", async () => {
  mockedGetAuthSession.mockResolvedValue({
    enabled: false,
    authenticated: true,
  });

  renderAuthGate();

  expect(await screen.findByText("Dashboard for anonymous")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Test sign out" })).not.toBeInTheDocument();
});

it("shows the server's generic authentication error", async () => {
  mockedGetAuthSession.mockResolvedValue({
    enabled: true,
    authenticated: false,
  });
  mockedLogin.mockRejectedValue({
    isAxiosError: true,
    response: { data: { error: "Invalid username or password." } },
  });

  renderAuthGate();

  await userEvent.type(await screen.findByLabelText("Username"), "operator");
  await userEvent.type(screen.getByLabelText("Password"), "wrong");
  await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

  expect(await screen.findByRole("alert")).toHaveTextContent(
    "Invalid username or password."
  );
});

it("registers a handler that returns expired sessions to sign in", async () => {
  mockedGetAuthSession.mockResolvedValue({
    enabled: true,
    authenticated: true,
    username: "operator",
    csrfToken: "csrf-token",
  });

  renderAuthGate();
  expect(await screen.findByText("Dashboard for operator")).toBeInTheDocument();

  const handlerCall = mockedSetUnauthorizedHandler.mock.calls.find(
    ([handler]) => typeof handler === "function"
  );
  if (!handlerCall || !handlerCall[0]) {
    throw new Error("unauthorized handler was not registered");
  }
  handlerCall[0]();

  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeInTheDocument();
});
