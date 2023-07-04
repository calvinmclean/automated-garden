<script lang="ts">
    import { onMount } from "svelte";
    import Gardens from "../components/Gardens.svelte";
    import { getGardens } from "../lib/gardenClient";
    import type { GardenResponse } from "../lib/gardenClient";

    let gardens: GardenResponse[];

    onMount(async () => {
        await getGardens(true)
            .then((response) => response.data)
            .then((data) => {
                gardens = data.gardens;
            });
    });
</script>

<h1>Gardens</h1>
<Gardens {gardens} />
