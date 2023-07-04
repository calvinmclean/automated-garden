<script lang="ts">
    import Zones from "../zones/Zones.svelte";
    import type { GardenResponse } from "../../lib/gardenClient";

    export let garden: GardenResponse;
</script>

{#if garden}
    <h1>{garden.name}</h1>

    <Zones gardenID={garden.id} />
{/if}
