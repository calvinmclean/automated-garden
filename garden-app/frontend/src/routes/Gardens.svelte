<script lang="ts">
    import Gardens from "../components/gardens/Gardens.svelte";
    import type { GardenResponse } from "../lib/gardenClient";
    import { gardenStore } from "../store";

    let gardens: GardenResponse[];

    gardenStore.init();
    gardenStore.subscribe((value) => {
        gardens = value;
    });
</script>

<Gardens {gardens} />
