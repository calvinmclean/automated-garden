<script lang="ts">
    import { onMount } from "svelte";
    import Garden from "../components/gardens/Garden.svelte";
    import { getGarden, type GardenResponse, type GetGardenParams } from "../lib/gardenClient";
    import { gardenStore } from "../store";

    export let params: GetGardenParams;

    let garden: GardenResponse;

    gardenStore.subscribe((value) => {
        garden = gardenStore.getByID(value, params.gardenID);
    });
</script>

<Garden {garden} />
