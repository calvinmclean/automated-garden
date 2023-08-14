import { writable } from 'svelte/store';
import { getGardens, type GardenResponse, type AllGardensResponse } from "./lib/gardenClient";
import { getZones, type ZoneResponse, type AllZonesResponse } from "./lib/zoneClient";


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
                gardens: data.gardens,
                zoneStores: {}
            };

            // Initialize zoneStores for each garden
            data.gardens.forEach(garden => {
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
    zoneStores[gardenID].set({ loading: true, zones: data.zones });

    response = await getZones(gardenID, true, false);
    data = response.data;
    zoneStores[gardenID].set({ loading: false, zones: data.zones });
};