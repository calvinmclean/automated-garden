<script lang="ts">
    import { Button } from "sveltestrap";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import { waterZone } from "../../lib/zoneClient";
    import ZoneCard from "./ZoneCard.svelte";

    export let gardenID: string;
    export let zone: ZoneResponse;

    let a = 1;

    function onClick(event) {
        console.log(`WATERING FOR ${a}`);
        waterZone(gardenID, zone.id, a);
    }
</script>

{#if zone}
    <ZoneCard {zone} />

    <div>
        <input type="number" bind:value={a} min="0" /> minutes
        <Button on:click={onClick} color={"primary"}>Water!</Button>
    </div>
{/if}
