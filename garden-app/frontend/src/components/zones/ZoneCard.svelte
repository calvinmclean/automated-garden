<script lang="ts">
    import {
        Badge,
        Button,
        ButtonDropdown,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        DropdownToggle,
        DropdownMenu,
        DropdownItem,
        Icon,
        Spinner,
    } from "sveltestrap";
    import { fly } from "svelte/transition";
    import { location } from "svelte-spa-router";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import { endDateZone, restoreZone } from "../../lib/zoneClient";

    export let gardenID: string;
    export let zone: ZoneResponse;
    export let withLink = true;
    export let loadingWeatherData = false;

    function deleteZone(event) {
        zone.end_date = Date.now().toLocaleString();
        endDateZone(gardenID, zone.id);
    }

    function onRestore(event) {
        zone.end_date = null;
        restoreZone(gardenID, zone.id);
    }
</script>

<div in:fly={{ x: 50, duration: 500 }} out:fly={{ x: -50, duration: 250 }}>
    <Card class=".col-lg-4" style="margin: 5%">
        {#if withLink}
            <a href="#{$location}/zones/{zone.id}">
                <CardHeader>
                    <CardTitle>
                        {zone.name}
                        {#if loadingWeatherData}
                            <Spinner color={"success"} type="border" />
                        {/if}
                    </CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>
                    {zone.name}
                    {#if loadingWeatherData}
                        <Spinner color={"success"} type="border" />
                    {/if}
                </CardTitle>
            </CardHeader>
        {/if}
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
                        Rain Scale Factor: {zone.weather_data.rain
                            .scale_factor}<br />
                    {/if}
                    {#if zone.weather_data.average_temperature != null}
                        Average High Temp ºF: {zone.weather_data
                            .average_temperature.celsius *
                            1.8 +
                            32}<br />
                        Average High Temp Scale Factor: {zone.weather_data
                            .average_temperature.scale_factor}<br />
                    {/if}
                {/if}

                <ButtonDropdown>
                    <DropdownToggle color="danger" caret>
                        <Icon name="trash" />
                    </DropdownToggle>
                    <DropdownMenu>
                        <DropdownItem on:click={deleteZone}>
                            {#if zone.end_date == null}
                                Confirm Delete
                            {:else}
                                Permanently Delete
                            {/if}
                        </DropdownItem>
                    </DropdownMenu>
                </ButtonDropdown>

                {#if zone.end_date != null}
                    <div>
                        <Button color="primary" on:click={onRestore}>
                            <Icon name="arrow-clockwise" />
                        </Button>
                    </div>
                {/if}
            </CardText>
        </CardBody>
        <CardFooter>
            {#if zone.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}
        </CardFooter>
    </Card>
</div>
