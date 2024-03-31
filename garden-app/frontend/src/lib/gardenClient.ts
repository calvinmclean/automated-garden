import createClient from "openapi-fetch";
import type { FetchResponse } from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: process.env.NODE_ENV == "docker" ? "" : "http://localhost:8080",
});

// types
export type GardenResponse = components["schemas"]["GardenResponse"];
export type AllGardensResponse = components["schemas"]["AllGardensResponse"];
export type GetGardenParams = operations["getGarden"]["parameters"]["path"];

let mockGardens: AllGardensResponse = {
    items: [
        {
            name: "Front Yard",
            id: "chokmn1nhf81274ru2mg",
            created_at: new Date().toISOString(),
            end_date: null,
            health: {
                status: "UP",
                details: "last contact from Garden was 30s ago"
            },
            topic_prefix: "front-yard",
            max_zones: 3,
            num_zones: 2,
            light_schedule: null,
            next_light_action: null,
            temperature_humidity_data: null,
            num_plants: 0,
            plants: null,
            zones: null,
            links: null,
        },
        {
            name: "Indoor Seed Starting",
            id: "cihl5mpnhf833c53rec0",
            created_at: new Date().toISOString(),
            end_date: null,
            health: {
                status: "UP",
                details: "last contact from Garden was 30s ago"
            },
            topic_prefix: "seed-garden",
            max_zones: 3,
            num_zones: 3,
            light_schedule: {
                duration: "16h0m0s",
                start_time: "06:00:00-07:00",
            },
            next_light_action: {
                time: new Date().toISOString(),
                state: "ON",
            },
            temperature_humidity_data: {
                temperature_celsius: 23.9,
                humidity_percentage: 40,
            },
            num_plants: 0,
            plants: null,
            zones: null,
            links: null,
        }
    ]
};

let demoMode = process.env.NODE_ENV == "demo";

// functions
export function getGardens(end_dated: boolean) {
    if (demoMode) {
        return new Promise<FetchResponse<AllGardensResponse>>(function (resolve, reject) {
            return resolve({
                data: mockGardens,
                response: new Response()
            })
        });
    }

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
    if (demoMode) {
        return new Promise<FetchResponse<GardenResponse>>(function (resolve, reject) {
            return resolve({
                data: mockGardens.items.find((g) => g.id == id),
                response: new Response()
            })
        });
    }

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

