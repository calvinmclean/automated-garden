import createClient from "openapi-fetch";
import type { paths, components, operations } from "./schema";

const { get, post, put, patch, del } = createClient<paths>({
    baseUrl: process.env.NODE_ENV == "docker" ? "" : "http://localhost:8080",
});

// types
export type AllWaterSchedulesResponse = components["schemas"]["AllWaterSchedulesResponse"];
export type WaterScheduleResponse = components["schemas"]["WaterScheduleResponse"];
export type GetWaterScheduleParams = operations["getWaterSchedule"]["parameters"]["path"];
export type WeatherData = components["schemas"]["WeatherData"]
export type NextWaterDetails = components["schemas"]["NextWaterDetails"]

// functions
export function getWaterSchedules(end_dated: boolean, exclude_weather_data: boolean) {
    return get("/water_schedules", {
        params: {
            query: {
                end_dated: end_dated,
                exclude_weather_data: exclude_weather_data,
            }
        },
        body: undefined as never,
    });
}

export function getWaterSchedule(id: string, exclude_weather_data: boolean) {
    return get("/water_schedules/{waterScheduleID}", {
        params: {
            path: {
                waterScheduleID: id,
            },
            query: {
                exclude_weather_data: exclude_weather_data,
            }
        },
        body: undefined as never,
    });
}
