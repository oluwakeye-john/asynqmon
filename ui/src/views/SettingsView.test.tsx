import React from "react";
import { Provider } from "react-redux";
import { render, screen } from "@testing-library/react";
import store from "../store";
import SettingsView from "./SettingsView";

it("renders the settings controls", () => {
  render(
    <Provider store={store}>
      <SettingsView />
    </Provider>
  );

  expect(screen.getByRole("heading", { name: "Settings" })).toBeInTheDocument();
  expect(screen.getByText("Polling Interval")).toBeInTheDocument();
  expect(screen.getByText("Dark Theme")).toBeInTheDocument();
});
