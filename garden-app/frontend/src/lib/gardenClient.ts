import createClient from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: "http://localhost:8080",
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
