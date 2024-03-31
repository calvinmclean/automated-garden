import { writable } from 'svelte/store';
import { getGardens, type GardenResponse, type AllGardensResponse } from "./lib/gardenClient";
import { getZones, type AllZonesResponse } from "./lib/zoneClient";
import { getWaterSchedules, type WaterScheduleResponse, type AllWaterSchedulesResponse } from "./lib/waterScheduleClient"


interface gardenStore {
    gardens: GardenResponse[];
    zoneStores: Record<string, ReturnType<typeof createZoneStore>>;
}

const createGardenStore = () => {
    const { subscribe, set } = writable<gardenStore>({
        gardens: [],
        zoneStores: {}
    });

    return {
        subscribe,
        init: async () => {
            const response = await getGardens(true);
            const data: AllGardensResponse = response.data;

            const gardenStoreData: gardenStore = {
                gardens: data.items,
                zoneStores: {}
            };

            // Initialize zoneStores for each garden
            data.items.forEach(garden => {
                gardenStoreData.zoneStores[garden.id] = createZoneStore(garden.id);
            });

            set(gardenStoreData);
        },
        getByID: (self: gardenStore, gardenID: string): GardenResponse => {
            return self.gardens.find(g => g.id == gardenID);
        }
    };
};

export const gardenStore = createGardenStore();

const zoneStores = {};

export const createZoneStore = (gardenID) => {
    if (!zoneStores[gardenID]) {
        zoneStores[gardenID] = writable({
            loading: true,
            zones: []
        });
        initZoneStore(gardenID);
    }

    return zoneStores[gardenID];
};

const initZoneStore = async (gardenID) => {
    let response = await getZones(gardenID, true, true);
    let data: AllZonesResponse = response.data;
    zoneStores[gardenID].set({ loading: true, zones: data.items });

    response = await getZones(gardenID, true, false);
    data = response.data;
    zoneStores[gardenID].set({ loading: false, zones: data.items });
};

interface waterscheduleStore {
    waterSchedules: WaterScheduleResponse[];
    loading: boolean;
}

const createWaterScheduleStore = () => {
    const { subscribe, set } = writable<waterscheduleStore>({
        waterSchedules: [],
        loading: true,
    });

    return {
        subscribe,
        init: async () => {
            let response = await getWaterSchedules(true, true);
            let data: AllWaterSchedulesResponse = response.data;;
            set({ waterSchedules: data.items, loading: true });

            response = await getWaterSchedules(true, false);
            data = response.data;;
            set({ waterSchedules: data.items, loading: false });
        },
        getByID: (self: waterscheduleStore, waterscheduleID: string): WaterScheduleResponse => {
            return self.waterSchedules.find(g => g.id == waterscheduleID);
        }
    };
};

export const waterScheduleStore = createWaterScheduleStore();
