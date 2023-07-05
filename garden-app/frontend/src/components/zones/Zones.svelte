<script lang="ts">
    import { onMount } from "svelte";
    import { getZones } from "../../lib/zoneClient";
    import ZoneLink from "./ZoneLink.svelte";
    import type { ZoneResponse } from "../../lib/zoneClient";

    export let gardenID: string;

    let zones: ZoneResponse[];

    // quickly get zones
    onMount(async () => {
        await getZones(gardenID, true, true)
            .then((response) => response.data)
            .then((data) => {
                zones = data.zones;
            });
    });
    // then get with full details
    onMount(async () => {
        await getZones(gardenID, true, false)
            .then((response) => response.data)
            .then((data) => {
                zones = data.zones;
            });
    });
</script>

{#if zones}
    {#each zones as zone}
        <ZoneLink {zone} />
    {/each}
{:else}
    loading...
{/if}
