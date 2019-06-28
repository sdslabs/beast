<template>
  <div>
    <div class="beast">
      <p>BEAST</p>
    </div>
    <nav v-if="isUser">
      <router-link class="navbar" v-for="routes in links"
      v-bind:key="routes.id"
      :to="`${routes.page}`">{{routes.text}}</router-link>
    </nav>
    <nav v-if="isAdmin">
      <router-link class="navbar" v-for="routes in links"
      v-bind:key="routes.id"
      :to="`${routes.admin_page}`">{{routes.text}}</router-link>
    </nav>
    <nav>
      <router-link v-for="routes in links.slice(0,5)"
      v-bind:key="routes.id"
      :class="'navbar-' + (routes.id)"
      :to="`${routes.page}`">{{routes.text}}</router-link>
    </nav>
    <hr class="navbar-hr">
  </div>
</template>

<script>
export default {
  name: 'Navigation',
  data () {
    return {
      links: [
        {
          id: 0,
          text: 'Home',
          page: '/Home',
          admin_page: '/admin/Home'
        },
        {
          id: 1,
          text: 'Leaderboard',
          page: '/Leaderboard',
          admin_page: '/admin/Leaderboard'
        },
        {
          id: 2,
          text: 'Users',
          page: '/Users',
          admin_page: '/admin/Users'
        },
        {
          id: 3,
          text: 'Challenges',
          page: '/Challenges',
          admin_page: '/admin/Challenges'
        },
        {
          id: 4,
          text: 'Notification',
          page: '/Notification',
          admin_page: '/admin/Notification'
        },
        {
          id: 5,
          text: 'Settings',
          page: '/Settings',
          admin_page: '/admin/Settings'
        },
        {
          id: 6,
          text: 'Logout',
          page: '/Logout',
          admin_page: '/admin/Logout'
        }
      ]
    }
  }
}
</script>

<style lang="css">
@import '../assets/css/navbar.css';
</style>
