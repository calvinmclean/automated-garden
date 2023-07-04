<script lang="ts">
    import { onMount } from "svelte";
    import Gardens from "../components/Gardens.svelte";
    import type { components } from "../types/garden-app-openapi";

    type GardenResponse = components["schemas"]["GardenResponse"];
    type AllGardensResponse = components["schemas"]["AllGardensResponse"];

    let gardens: GardenResponse[];

    onMount(async () => {
        await fetch(`http://localhost:8080/gardens`)
            .then((response) => response.json() as Promise<AllGardensResponse>)
            .then((allGardens) => {
                gardens = allGardens.gardens;
            });
    });
</script>

<h1>Gardens</h1>
<Gardens {gardens} />
