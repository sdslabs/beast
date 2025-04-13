const puppeteer = require('puppeteer');
const BASE_URL = process.env.BASE_URL || 'http://0.0.0.0';
const FLAG= process.env.FLAG || 'test{flag}';
const sleep = ms => new Promise(res => setTimeout(res, ms));

const CONFIG = {
    APPNAME: process.env['APPNAME'] || "Admin",
    APPURL: process.env['BASE_URL'] || "http://127.0.0.1:8000",
    APPURLREGEX: process.env['APPURLREGEX'] || "^.*$",
    APPFLAG: process.env['FLAG'] || "winterhack{w45_th15_y0ur_f1r5t_4dm1n_b07_ch4ll}",
    APPLIMITTIME: Number(process.env['APPLIMITTIME'] || "60"),
    APPLIMIT: Number(process.env['APPLIMIT'] || "5"),
}

console.table(CONFIG)

const initBrowser = puppeteer.launch({
    executablePath: "/usr/bin/chromium-browser",
    headless: 'new',
    args: [
        '--disable-dev-shm-usage',
        '--no-sandbox',
        '--disable-setuid-sandbox',
        '--disable-gpu',
        '--no-gpu',
        '--disable-default-apps',
        '--disable-translate',
        '--disable-device-discovery-notifications',
        '--disable-software-rasterizer',
        '--disable-xss-auditor'
    ],
    ipDataDir: '/home/bot/data/',
    ignoreHTTPSErrors: true
});

console.log("Bot started...");

module.exports = {
    name: CONFIG.APPNAME,
    urlRegex: CONFIG.APPURLREGEX,
    rateLimit: {
        windowS: CONFIG.APPLIMITTIME,
        max: CONFIG.APPLIMIT
    },
    bot: async (urlToVisit) => {
        const browser = await initBrowser;
        const context = await browser.createIncognitoBrowserContext()
        try {
            const page = await context.newPage();
            url = BASE_URL ;
            await page.setCookie({ name: 'flag', value: FLAG, domain: new URL(url).hostname, httpOnly: false, secure: false });

            console.log(`bot visiting ${urlToVisit}`)
            await page.goto(urlToVisit, {
                timeout: 3000,
                waitUntil: 'domcontentloaded'
            });

            console.log("bot visited")
            await sleep(20000)

            console.log("browser close...")
            await context.close()
            return true;
        } catch (e) {
            console.error(e);
            await context.close();
            return false;
        }
    }
}