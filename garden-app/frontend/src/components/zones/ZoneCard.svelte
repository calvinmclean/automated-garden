<script lang="ts">
    import {
        Badge,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        Icon,
    } from "sveltestrap";
    import { location } from "svelte-spa-router";
    import type { ZoneResponse } from "../../lib/zoneClient";

    export let zone: ZoneResponse;
    export let withLink = false;
</script>

<Card class=".col-lg-4" style="margin: 5%">
    <CardHeader>
        {#if withLink}
            <a href="#{$location}/zones/{zone.id}">
                <CardTitle>{zone.name}</CardTitle>
            </a>
        {:else}
            <CardTitle>{zone.name}</CardTitle>
        {/if}
    </CardHeader>
    <CardBody>
        <CardSubtitle>{zone.id}</CardSubtitle>
        <CardText>
            {#if zone.end_date != null}
                End Dated: {zone.end_date}
                <Icon name="clock-fill" style="color: red" /><br />
            {/if}

            {#if zone.skip_count != null}
                Skip Count: {zone.skip_count}<br />
            {/if}

            {#if zone.next_water != null}
                Next Water Time: {zone.next_water.time}
                <Icon name="clock" /><br />
                Next Water Duration: {zone.next_water.duration}<br />
                Next Water Message: {zone.next_water.message}<br />
            {/if}

            {#if zone.weather_data != null}
                {#if zone.weather_data.rain != null}
                    Rain MM: {zone.weather_data.rain.mm}<br />
                    Rain Scale Factor: {zone.weather_data.rain.scale_factor}<br
                    />
                {/if}
                {#if zone.weather_data.average_temperature != null}
                    Average High Temp ºF: {zone.weather_data.average_temperature
                        .celsius *
                        1.8 +
                        32}<br />
                    Average High Temp Scale Factor: {zone.weather_data
                        .average_temperature.scale_factor}<br />
                {/if}
            {/if}
        </CardText>
    </CardBody>
    <CardFooter>
        {#if zone.end_date != null}
            <Badge color={"danger"}>End Dated</Badge>
        {/if}
    </CardFooter>
</Card>
