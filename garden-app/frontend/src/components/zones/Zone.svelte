<script lang="ts">
    import { Button } from "sveltestrap";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import { waterZone } from "../../lib/zoneClient";
    import ZoneCard from "./ZoneCard.svelte";

    export let gardenID: string;
    export let zone: ZoneResponse;
    export let loadingWeatherData = false;

    let minutes = 1;

    function sendWaterRequest(event) {
        waterZone(gardenID, zone.id, minutes);
    }
</script>

{#if zone}
    <ZoneCard {gardenID} {zone} {loadingWeatherData} withLink={false} />

    <div>
        <input type="number" bind:value={minutes} min="0" /> minutes
        <Button on:click={sendWaterRequest} color={"primary"}>Water!</Button>
    </div>
{/if}