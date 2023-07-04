<script lang="ts">
    import { Button } from "sveltestrap";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import { waterZone } from "../../lib/zoneClient";

    export let gardenID: string;
    export let zone: ZoneResponse;

    let a = 1;

    function onClick(event) {
        console.log(`WATERING FOR ${a}`);
        waterZone(gardenID, zone.id, a);
    }
</script>

{#if zone}
    <h1>{zone.name}</h1>

    <div>
        <input type="number" bind:value={a} min="0" /> minutes
        <Button on:click={onClick} color={"primary"}>Water!</Button>
    </div>
{/if}
