import { writable } from 'svelte/store';
import { getGardens, type AllGardensResponse } from "./lib/gardenClient";
import { getZone, getZones, type ZoneResponse, type AllZonesResponse } from "./lib/zoneClient";

const createGardenStore = () => {
    const { subscribe, set } = writable([])

    return {
        subscribe,
        init: async () => {
            const response = await getGardens(true);
            const data: AllGardensResponse = response.data;
            set(data.gardens);
        }
    }
};

export const gardenStore = createGardenStore();

const createZoneStore = () => {
    const { subscribe, set } = writable({ loading: true, zones: [] })

    return {
        subscribe,
        init: async (gardenID: string) => {
            // Get without weather data included for fast response
            let response = await getZones(gardenID, true, true);
            let data: AllZonesResponse = response.data;
            set({ loading: true, zones: data.zones });

            // Get with weather data included
            response = await getZones(gardenID, true, false);
            data = response.data;
            set({ loading: false, zones: data.zones });
        }
    }
};

export const zoneStore = createZoneStore();
