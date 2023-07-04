import createClient from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: "http://localhost:8080",
});

// types
export type ZoneResponse = components["schemas"]["ZoneResponse"];
export type GetZoneParams = operations["getZone"]["parameters"]["path"];

// functions
export function getZones(gardenID: string, end_dated: boolean) {
    return get("/gardens/{gardenID}/zones", {
        params: {
            path: {
                gardenID: gardenID,
            },
            query: {
                end_dated: end_dated,
            }
        },
        body: undefined as never,
    });
}

export function getZone(gardenID: string, id: string) {
    return get("/gardens/{gardenID}/zones/{zoneID}", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: id,
            }
        },
        body: undefined as never,
    });
}
