import "@testing-library/jest-dom/vitest";

// Material UI v4/JSS invokes CSS.escape as an unbound function. jsdom's
// generated implementation requires a CSS receiver, unlike browsers.
Object.defineProperty(globalThis.CSS, "escape", {
  configurable: true,
  value: (value: string) =>
    String(value).replace(/[^a-zA-Z0-9_-]/g, (character) =>
      `\\${character.codePointAt(0)?.toString(16)} `
    ),
});
