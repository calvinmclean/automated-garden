<script>
    import { onMount } from "svelte";
    import Garden from "./Garden.svelte";

    let gardens;

    onMount(async () => {
        await fetch(`http://localhost:8080/gardens`)
            .then((r) => r.json())
            .then((data) => {
                gardens = data.gardens;
            });
    });
</script>

{#if gardens}
    {#each gardens as garden}
        <ul>
            <li>
                <Garden {garden} />
            </li>
        </ul>
    {/each}
{:else}
    <p class="loading">loading...</p>
{/if}
