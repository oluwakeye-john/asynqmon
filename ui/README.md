# Working with the React UI

This file explains how to work with Asynqmon UI.

## Introduction

The Asynqmon UI uses [Vite](https://vite.dev/) for local development, type checking, testing, and production builds.

Instead of plain JavaScript, we use [TypeScript](https://www.typescriptlang.org/) to ensure typed code.

## Development environment

To work with the React UI code, you will need to have the following tools installed:

- The [Node.js](https://nodejs.org/) JavaScript runtime, version 20.19 or newer. Node.js 24 is the version used by CI and Docker builds.
- The [Yarn](https://yarnpkg.com/) package manager.
- _Recommended:_ An editor with TypeScript and React support. If you are not sure which editor to use, we recommend using [Visual Studio Code](https://code.visualstudio.com/docs/languages/typescript). Make sure that [the editor uses the project's TypeScript version rather than its own](https://code.visualstudio.com/docs/typescript/typescript-compiling#_using-the-workspace-version-of-typescript).

**NOTE**: When using Visual Studio Code, be sure to open the `ui/` directory in the editor instead of the root of the repository. This way, the right ESLint and TypeScript configuration will be picked up from the React workspace.

## Installing npm dependencies

The React UI depends on a large number of [npm](https://www.npmjs.com/) packages. These are not checked in, so you will need to download and install them locally via the Yarn package manager:

    yarn

Yarn consults the `package.json` and `yarn.lock` files for dependencies to install. It creates a `node_modules` directory with all installed dependencies.

**NOTE**: Remember to change directory to `ui/` before running this command and the following commands.

## Running a local development server

You can start a development server for the React UI outside of a running Asynqmon server by running:

    yarn start

This starts the Vite development server at http://localhost:3000/. The page reloads when you edit the source code.

The development UI expects the Asynqmon API server at http://localhost:8080/.

## Running tests

Run the Vitest suite once with:

    yarn test

Use `yarn test:watch` while developing.

## Building the app for production

To type-check the UI and create a production-optimized bundle in the `build` subdirectory, run:

    yarn build

**NOTE:** You will likely not need to do this directly. Instead, this is taken care of by the `build` target in the main Asynqmon `Makefile` when building the full binary.

## Integration into Asynqmon

To build a Asynqmon binary that includes a compiled-in version of the production build of the React app, change to the root of the repository and run:

    make build

This installs npm dependencies via Yarn, builds a production build of the React app, and then finally compiles in all web assets into the Asynqmon binary.
