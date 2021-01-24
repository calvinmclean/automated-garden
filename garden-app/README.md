# Garden App

This is a Go application with a CLI and web backend for working with the garden controller.


## Getting Started
WIP


## Design Choices

### Code Organization
The base of this project is a [Cobra](https://github.com/spf13/cobra) CLI application. It is used to start up a [`go-chi`](https://github.com/go-chi/chi) web application.

Currently, this consists of 3 base packages:
- `api`: contains models and other core code necessary for the application's functionality
- `cmd`: contains Cobra commands for working with the other packages
- `http`: contains most of the `go-chi` parts of the application for implementing the HTTP API

This approach allows me to focus on the application's core functionality separate from how the user will interact with it through the CLI or HTTP API.
