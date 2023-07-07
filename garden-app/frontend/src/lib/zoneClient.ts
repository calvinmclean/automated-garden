import createClient from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: "",
});

// types
export type ZoneResponse = components["schemas"]["ZoneResponse"];
export type GetZoneParams = operations["getZone"]["parameters"]["path"];

// functions
export function getZones(gardenID: string, end_dated: boolean, exclude_weather_data: boolean) {
    return get("/gardens/{gardenID}/zones", {
        params: {
            path: {
                gardenID: gardenID,
            },
            query: {
                end_dated: end_dated,
                exclude_weather_data: exclude_weather_data,
            },
        },
        body: undefined as never,
    });
}

export function getZone(gardenID: string, id: string, exclude_weather_data: boolean) {
    return get("/gardens/{gardenID}/zones/{zoneID}", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: id,
            },
            query: {
                exclude_weather_data: exclude_weather_data,
            },
        },
        body: undefined as never,
    });
}

export function endDateZone(gardenID: string, id: string) {
    return del("/gardens/{gardenID}/zones/{zoneID}", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: id,
            },
        },
        body: undefined as never,
    });
}

export function restoreZone(gardenID: string, id: string) {
    return patch("/gardens/{gardenID}/zones/{zoneID}", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: id,
            },
        },
        body: {
            end_date: null,
        },
    });
}

export function waterZone(gardenID: string, zoneID: string, minutes: number) {
    return post("/gardens/{gardenID}/zones/{zoneID}/action", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: zoneID,
            }
        },
        body: {
            water: {
                duration: `${minutes}m`
            }
        },
    });
}
