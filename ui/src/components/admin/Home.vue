<template>
  <div class="home_div">
    <div class="competition_details_div">
      <div class="competition_name">
        {{ competition.name }}
      </div>
      <div class="competition_desc">
        {{ competition.desc }}
      </div>
      <div class="competition_date">
        {{ competition.date }}
      </div>
      <div class="competition_time">
        {{ competition.time }}
      </div>
      <div class="remaining_time_to_start">
        in &nbsp; <span class="days_remaining"> {{ days }} </span> &nbsp; days
      </div>
    </div>
    <div class="see_challenge_button_div">
      <button class="challenge_button" @click="$router.push('/Challenges')"><p class="see_challenges">See Challenges</p></button>
    </div>
    <div class="rules_and_regulations">
      <div class="heading">
        Rules and Regulations
      </div>
      <div class="rules">
        <p class="main_rules">Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.</p>
        <ul class="rules_list">
          <li>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</li>
          <li>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</li>
          <li>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Home',
  data () {
    return {
      competition: {
        name: 'n00bCTF 2019',
        desc: 'A beginner level CTF, especially designed for first years. Top two participants from the first year will get a wild card entry to the final round of SDSLabs interview. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat',
        date: '10th January 2019, Wednesday',
        time: '16:00 - 22:00 IST'
      },
      start: '',
      interval: '',
      minutes: '',
      hours: '',
      seconds: '',
      starttime: 'Nov 5, 2019 15:37:25'
    }
  },

  mounted () {
    this.start = new Date(this.starttime).getTime()
    // Update the count down every 1 second
    this.timerCount(this.start)
    this.interval = setInterval(() => {
      this.timerCount(this.start)
    }, 1000)
  },

  methods: {
    timerCount: function (start) {
      var now = new Date().getTime()
      var diff = start - now
      this.calcTime(diff)
    },
    calcTime: function (diff) {
      this.days = Math.floor(diff / (1000 * 60 * 60 * 24))
      this.hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
      this.minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
      this.seconds = Math.floor((diff % (1000 * 60)) / 1000)
    }
  }
}
</script>

<style lang="css">
@import '../../assets/css/home.css';
</style>
