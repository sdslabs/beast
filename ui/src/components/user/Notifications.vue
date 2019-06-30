<template>
  <div class="notification_div">
    <div v-for="notify in notification" v-bind:key="notify.id" class="notification_box">
      <div class="notification_title">
        {{notify.title}}
      </div>
      <div class="notification_timing">
        {{notify.timing}}
      </div>
      <div class="notification_desc">
        {{notify.desc}}
      </div>
      <div v-if="notify.status" class="notification_status">
        <div class="notification_status_new">{{isNew(notify)}}</div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Notifications',
  data () {
    return {
      notification: [
        {
          id: 1,
          title: 'CTF is live now',
          desc: 'n00b2019 is live! Now you can access challanges and start playing.',
          timing: '5 min ago',
          status: 1
        },
        {
          id: 2,
          title: 'CTF is live now',
          desc: 'n00b2018 is live! Now you can access challanges and start playing.',
          timing: '1 year 5 min ago',
          status: 1
        },
        {
          id: 3,
          title: 'CTF is live now',
          desc: 'n00b2017 is live! Now you can access challanges and start playing.',
          timing: '2 year 5 min ago',
          status: 0
        },
        {
          id: 4,
          title: 'CTF is live now',
          desc: 'n00b2017 is live! Now you can access challanges and start playing.',
          timing: '2 year 5 min ago',
          status: 0
        }
      ]
    }
  },
  methods: {
    isNew: function (notification) {
      if (notification.status === 1) {
        return 'NEW'
      }
    }
  }
}
</script>

<style lang="css">
@import '../../assets/css/notifications.css';
</style>
