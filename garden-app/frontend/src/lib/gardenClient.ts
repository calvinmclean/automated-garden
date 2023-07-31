import createClient from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: process.env.NODE_ENV == "docker" ? "" : "http://localhost:8080",
});

// types
export type GardenResponse = components["schemas"]["GardenResponse"];
export type GetGardenParams = operations["getGarden"]["parameters"]["path"];

// functions
export function getGardens(end_dated: boolean) {
    return get("/gardens", {
        params: {
            query: {
                end_dated: end_dated,
            }
        },
        body: undefined as never,
    });
}

export function getGarden(id: string) {
    return get("/gardens/{gardenID}", {
        params: {
            path: {
                gardenID: id,
            }
        },
        body: undefined as never,
    });
}

export function lightAction(id: string, state: "ON" | "OFF" | "") {
    return post("/gardens/{gardenID}/action", {
        params: {
            path: {
                gardenID: id,
            }
        },
        body: {
            light: {
                state: state
            }
        },
    });
}

export function stopAction(id: string) {
    return post("/gardens/{gardenID}/action", {
        params: {
            path: {
                gardenID: id,
            }
        },
        body: {
            stop: {
                all: true
            }
        },
    });
}

