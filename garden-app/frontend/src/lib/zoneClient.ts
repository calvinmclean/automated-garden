import createClient from "openapi-fetch";
import type { FetchResponse } from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: process.env.NODE_ENV == "docker" ? "" : "http://localhost:8080",
});

// types
export type ZoneResponse = components["schemas"]["ZoneResponse"];
export type AllZonesResponse = components["schemas"]["AllZonesResponse"];
export type GetZoneParams = operations["getZone"]["parameters"]["path"];

export type WaterHistoryResponse = components["schemas"]["WaterHistoryResponse"];

let mockZones = new Map<string, AllZonesResponse>()
mockZones.set("chokmn1nhf81274ru2mg", { // front-yard
    zones: [
        {
            name: "Trees",
            details: null,
            position: 0,
            skip_count: 1,
            water_schedule_ids: null,
            id: "cja66ba8tio4p0u075c0",
            created_at: new Date().toISOString(),
            end_date: null,
            next_water: {
                time: new Date().toISOString(),
                duration: "2h0m0s",
                water_schedule_id: null,
                message: "skip_count 1 affected the time"
            },
            weather_data: null,
            links: null,
        },
        {
            name: "Shrubs",
            details: null,
            position: 1,
            skip_count: null,
            water_schedule_ids: null,
            id: "cja66di8tio4p91qkbq0",
            created_at: new Date().toISOString(),
            end_date: null,
            next_water: {
                time: new Date().toISOString(),
                duration: "1h0m0s",
                water_schedule_id: null,
                message: null,
            },
            weather_data: null,
            links: null,
        }
    ]
})
mockZones.set("cihl5mpnhf833c53rec0", { // seed-garden
    zones: [
        {
            name: "Tray 1",
            details: null,
            position: null,
            skip_count: null,
            water_schedule_ids: null,
            id: "cja66ga8tio4pqckuu40",
            created_at: new Date().toISOString(),
            end_date: null,
            next_water: {
                time: new Date().toISOString(),
                duration: "30s",
                water_schedule_id: null,
                message: null,
            },
            weather_data: null,
            links: null,
        },
        {
            name: "Tray 2",
            details: null,
            position: null,
            skip_count: null,
            water_schedule_ids: null,
            id: "cja66ga8tio4pqckuu4g",
            created_at: new Date().toISOString(),
            end_date: null,
            next_water: {
                time: new Date().toISOString(),
                duration: "30s",
                water_schedule_id: null,
                message: null,
            },
            weather_data: null,
            links: null,
        },
        {
            name: "Tray 3",
            details: null,
            position: null,
            skip_count: null,
            water_schedule_ids: null,
            id: "cja66ga8tio4pqckuu50",
            created_at: new Date().toISOString(),
            end_date: null,
            next_water: {
                time: new Date().toISOString(),
                duration: "30s",
                water_schedule_id: null,
                message: null,
            },
            weather_data: null,
            links: null,
        }
    ]
})

let demoMode = process.env.NODE_ENV == "demo";

// functions
export function getZones(gardenID: string, end_dated: boolean, exclude_weather_data: boolean) {
    if (demoMode) {
        return new Promise<FetchResponse<AllZonesResponse>>(function (resolve, reject) {
            return resolve({
                data: mockZones.get(gardenID),
                response: new Response()
            })
        });
    }

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
    if (demoMode) {
        return new Promise<FetchResponse<AllZonesResponse>>(function (resolve, reject) {
            return resolve({
                data: mockZones.get(gardenID).zones.find((z) => z.id == id),
                response: new Response()
            })
        });
    }

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


export function getZoneWaterHistory(gardenID: string, zoneID: string, rangeDays: number, limit: number = 10) {
    return get("/gardens/{gardenID}/zones/{zoneID}/history", {
        params: {
            path: {
                gardenID: gardenID,
                zoneID: zoneID
            },
            query: {
                range: `${rangeDays * 24}h`,
                limit: limit
            },
        },
        body: undefined as never,
    });
}
