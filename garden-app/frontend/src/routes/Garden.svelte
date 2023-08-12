<script lang="ts">
    import { onMount } from "svelte";
    import Garden from "../components/gardens/Garden.svelte";
    import {} from "../lib/gardenClient";
    import { getGarden, type GardenResponse, type GetGardenParams } from "../lib/gardenClient";

    export let params: GetGardenParams;

    let garden: GardenResponse;

    onMount(async () => {
        await getGarden(params.gardenID)
            .then((response) => response.data)
            .then((data: GardenResponse) => {
                garden = data;
            });
    });
</script>

<Garden {garden} />
