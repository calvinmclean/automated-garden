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

interface zStore {
    zones: ZoneResponse[],
    loading: boolean,
    gardenID: string
};

const createZoneStore = () => {
    const { subscribe, set, } = writable(<zStore>{
        loading: true, zones: [], gardenID: ""
    });

    return {
        subscribe,
        init: async (gardenID: string) => {
            // Get without weather data included for fast response
            let response = await getZones(gardenID, true, true);
            let data: AllZonesResponse = response.data;
            set({ loading: true, zones: data.zones, gardenID: gardenID });

            // Get with weather data included
            response = await getZones(gardenID, true, false);
            data = response.data;
            set({ loading: false, zones: data.zones, gardenID: gardenID, });
        },
        getByID: (self: zStore, zoneID: string): ZoneResponse => {
            return self.zones.find(z => z.id == zoneID)
        }
    }
};

export const zoneStore = createZoneStore();
