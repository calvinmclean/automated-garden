import { writable } from 'svelte/store';
import { getGardens } from "./lib/gardenClient";
import type { GardenResponse } from "./lib/gardenClient";

const createGardenStore = () => {
    const { subscribe, update, set } = writable([])

    return {
        subscribe,
        init: async () => {
            const response = await getGardens(true);
            const data = response.data;
            set(data.gardens);
        }
    }
}

export const gardenStore = createGardenStore()
