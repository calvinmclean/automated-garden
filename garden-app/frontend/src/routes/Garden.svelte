<script lang="ts">
    import { onMount } from "svelte";
    import Garden from "../components/Garden.svelte";
    import { getGarden } from "../lib/gardenClient";
    import type { GardenResponse } from "../lib/gardenClient";

    export let params;

    let garden: GardenResponse;

    onMount(async () => {
        await getGarden(params.garden_id)
            .then((response) => response.data)
            .then((data) => {
                garden = data;
            });
    });
</script>

<Garden {garden} />
