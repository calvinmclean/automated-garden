<script lang="ts">
    import { onMount } from "svelte";
    import Garden from "../components/gardens/Garden.svelte";
    import { getGarden } from "../lib/gardenClient";
    import type { GardenResponse, GetGardenParams } from "../lib/gardenClient";

    export let params: GetGardenParams;

    let garden: GardenResponse;

    onMount(async () => {
        await getGarden(params.gardenID)
            .then((response) => response.data)
            .then((data) => {
                garden = data;
            });
    });
</script>

<Garden {garden} />
