/* eslint-disable */
import Vue from 'vue'
import Router from 'vue-router'
import User_Challenges from '@/components/user/Challenges'
import User_Leaderboard from '@/components/user/Leaderboard'
import User_Users from '@/components/user/Users'
import User_Home from '@/components/user/Home'
import User_Logout from '@/components/user/Logout'
import User_Notifications from '@/components/user/Notifications'
import User_Settings from '@/components/user/Settings'
import Admin_Challenges from '@/components/admin/Challenges'
import Admin_Leaderboard from '@/components/admin/Leaderboard'
import Admin_Users from '@/components/admin/Users'
import Admin_Home from '@/components/admin/Home'
import Admin_Logout from '@/components/admin/Logout'
import Admin_Notifications from '@/components/admin/Notifications'
import Admin_Settings from '@/components/admin/Settings'
import Login from '@/components/Login'

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/challenges',
      name: 'Challenges',
      component: User_Challenges
    },
    {
      path: '/leaderboard',
      name: 'Leaderboard',
      component: User_Leaderboard
    },
    {
      path: '/users',
      name: 'Users',
      component: User_Users
    },
    {
      path: '/home',
      name: 'Home',
      component: User_Home
    },
    {
      path: '/logout',
      name: 'Logout',
      component: User_Logout
    },
    {
      path: '/notifications',
      name: 'Notifications',
      component: User_Notifications
    },
    {
      path: '/settings',
      name: 'Settings',
      component: User_Settings
    },
    {
      path: '/admin/challenges',
      name: 'Challenges',
      component: Admin_Challenges
    },
    {
      path: '/admin/leaderboard',
      name: 'Leaderboard',
      component: Admin_Leaderboard
    },
    {
      path: '/admin/users',
      name: 'Users',
      component: Admin_Users
    },
    {
      path: '/admin/home',
      name: 'Home',
      component: Admin_Home
    },
    {
      path: 'admin/logout',
      name: 'Logout',
      component: Admin_Logout
    },
    {
      path: '/admin/notifications',
      name: 'Notifications',
      component: Admin_Notifications
    },
    {
      path: '/admin/settings',
      name: 'Settings',
      component: Admin_Settings
    },
    {
      path: '/login',
      name: 'Login',
      component: Login
    },
  ]
})
