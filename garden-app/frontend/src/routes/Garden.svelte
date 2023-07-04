<script lang="ts">
    import { onMount } from "svelte";
    import Garden from "../components/Garden.svelte";
    import type { components } from "../types/garden-app-openapi";

    type GardenResponse = components["schemas"]["GardenResponse"];

    export let params;

    let garden: GardenResponse;

    onMount(async () => {
        await fetch(`http://localhost:8080/gardens/` + params.garden_id)
            .then((response) => response.json() as Promise<GardenResponse>)
            .then((data) => {
                garden = data;
            });
    });
</script>

<Garden {garden} />
