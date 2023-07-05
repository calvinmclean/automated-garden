<script lang="ts">
    import {
        Collapse,
        Navbar,
        NavbarToggler,
        NavbarBrand,
        Nav,
        NavItem,
        NavLink,
    } from "sveltestrap";

    let isOpen = false;

    function handleUpdate(event) {
        isOpen = event.detail.isOpen;
    }
</script>

<Navbar color="success" light expand="md">
    <NavbarBrand href="#/gardens">Garden App</NavbarBrand>
    <NavbarToggler on:click={() => (isOpen = !isOpen)} />
    <Collapse {isOpen} navbar expand="md" on:update={handleUpdate}>
        <Nav class="ms-auto" navbar>
            <NavItem>
                <NavLink href="#/gardens">Gardens</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="#/water_schedules">Water Schedules</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="#/weather_clients">Weather Clients</NavLink>
            </NavItem>
        </Nav>
    </Collapse>
</Navbar>
