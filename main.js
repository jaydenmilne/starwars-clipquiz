import {getManifest, getClipUrl} from './clips.js';

// main divs
let MAIN_PAGE;
let QUIZ_PAGE;
let RESULTS_PAGE;

// main screen
let EASY_BUTTON;
let MEDIUM_BUTTON;
let HARD_BUTTON; 
let LEGEND_BUTTON;

// quiz
let PROGRESS;
let PLAY_BUTTON;
let VOLUME_SLIDER;

let PHANTOM_MENACE;
let ATTACK_CLONES;
let REVENGE_SITH;

let NEW_HOPE;
let EMPIRE;
let ROTJ;

// results
let RESULTS_HEADING;
let RESULTS_LIST;
let DURATION;
let AGAIN_BUTTON;
let MAIN_BUTTON;

const MAIN = "main", QUIZ = "quiz", RESULTS = "results";
let currentState = MAIN;

const NUM_QUESTIONS = 10;

let selections = new Array(NUM_QUESTIONS);
let toAsk;
let toAskAudio;
let currentIndex = 0;
let selectedDifficulty;
let startTime;

const idToDisplay = new Map([
    ["phantom-menace", "Episode I"],
    ["attack-clones", "Episode II"],
    ["revenge-sith", "Episode III"],
    ["new-hope", "Episode IV"],
    ["empire", "Episode V"],
    ["rotj", "Episode VI"]
])

function router(newState) {
    currentState = newState;

    window.history.pushState({}, '', `#${newState}`);

    QUIZ_PAGE.style.display = "none";
    RESULTS_PAGE.style.display = "none";
    MAIN_PAGE.style.display = "none";

    switch (newState) {
        case MAIN:
            MAIN_PAGE.style.display = "flex";
            break;
        case QUIZ:
            QUIZ_PAGE.style.display = "flex";
            break;
        case RESULTS:
            RESULTS_PAGE.style.display = "flex";
            break;
    }
}

function getRandom(arr, n) {
    var result = new Array(n),
        len = arr.length,
        taken = new Array(len);
    if (n > len)
        throw new RangeError("getRandom: more elements taken than available");
    while (n--) {
        var x = Math.floor(Math.random() * len);
        result[n] = arr[x in taken ? taken[x] : x];
        taken[x] = --len in taken ? taken[len] : len;
    }
    return result;
}

function setVolume() {
    const vol = VOLUME_SLIDER.value;
    toAskAudio.forEach((clip) => {
        clip.volume = vol;
    });
}

async function beginQuiz(difficultyBtnName) {
    const difficulty = difficultyBtnName.split('-')[0];

    console.log("starting quiz at difficulty", difficulty);
    selections = new Array(NUM_QUESTIONS);
    currentIndex = 0;
    updateProgress();
    selectedDifficulty = difficulty;

    startTime = new Date();

    let clips = await getManifest(difficulty);

    toAsk = getRandom(clips, NUM_QUESTIONS);
    toAskAudio = await Promise.all(toAsk.map(async (clip) => {
        const url = await getClipUrl(clip.file);
        return new Audio(url);
    }));

    // play the first one as soon as it is ready
    toAskAudio[0].addEventListener('loadeddata', () => {
        toAskAudio[0].play();
    })

    toAskAudio[0].play();

    router(QUIZ);
    
}

function updateProgress() {
    PROGRESS.innerText = `${(currentIndex + 1).toString()}/${NUM_QUESTIONS.toString()}`;
}

function movieSelect(movie) {
    toAskAudio[currentIndex].pause();
    selections[currentIndex] = movie;
    currentIndex += 1;

    updateProgress();
    if (currentIndex >= NUM_QUESTIONS) {
        showResults();
    } else {
        toAskAudio[currentIndex].play();
    }
}

function formatDuration(duration) {
    var sec_num = parseInt(duration, 10); // don't forget the second param
    var hours   = Math.floor(sec_num / 3600);
    var minutes = Math.floor((sec_num - (hours * 3600)) / 60);
    var seconds = sec_num - (hours * 3600) - (minutes * 60);

    if (hours   < 10) {hours   = "0"+hours;}
    if (minutes < 10) {minutes = "0"+minutes;}
    if (seconds < 10) {seconds = "0"+seconds;}
    return hours+':'+minutes+':'+seconds;
}

function showResults() {
    stopAll();
    while (RESULTS_LIST.firstChild) {
        RESULTS_LIST.firstChild.remove()
    }

    let numCorrect = 0;
    for (let i = 0; i < NUM_QUESTIONS; ++i) {
        const li = document.createElement('li');
        const btn = document.createElement('button');
        const span = document.createElement('span');
        btn.className = "mini-play";
        btn.addEventListener("click", () => {
            stopAll();
            toAskAudio[i].play();
        });
        btn.innerText = '▶';
        li.appendChild(btn);

        if (selections[i] == toAsk[i].episode) {
            span.innerText = `✅ ${idToDisplay.get(selections[i])}`;
            ++numCorrect;
        } else {
            span.innerText = `❌ ${idToDisplay.get(toAsk[i].episode)} (not ${idToDisplay.get(selections[i])})`;
        }
        li.appendChild(span)
        RESULTS_LIST.appendChild(li)
    }

    RESULTS_HEADING.innerText = `Results (${selectedDifficulty}): ${numCorrect}/${NUM_QUESTIONS}`

    const timeTaken = new Date() - startTime;

    DURATION.innerText = formatDuration(timeTaken / 1000);
    router(RESULTS);
}

function stopAll() {
    for (let i = 0; i < NUM_QUESTIONS; ++i) {
        toAskAudio[i].pause();
        toAskAudio[i].currentTime = 0;
    }
}

function main() {
    EASY_BUTTON = document.getElementById("easy-button");
    MEDIUM_BUTTON = document.getElementById("medium-button");
    HARD_BUTTON = document.getElementById("hard-button");
    LEGEND_BUTTON = document.getElementById("legend-button");

    MAIN_PAGE = document.getElementById("main");
    QUIZ_PAGE = document.getElementById("quiz");
    RESULTS_PAGE = document.getElementById("results");

    PLAY_BUTTON = document.getElementById("play-button");
    VOLUME_SLIDER = document.getElementById("volume-slider");
    PHANTOM_MENACE = document.getElementById("phantom-menace");
    ATTACK_CLONES = document.getElementById("attack-clones");
    REVENGE_SITH = document.getElementById("revenge-sith");

    NEW_HOPE = document.getElementById("new-hope");
    EMPIRE = document.getElementById("empire");
    ROTJ = document.getElementById("rotj");

    PROGRESS = document.getElementById("progress");

    RESULTS_HEADING = document.getElementById("results-heading");
    RESULTS_LIST = document.getElementById("results-list");
    DURATION = document.getElementById("time-div");
    AGAIN_BUTTON = document.getElementById("again");
    MAIN_BUTTON = document.getElementById("change-difficulty");


    // add listeners

    [EASY_BUTTON, MEDIUM_BUTTON, HARD_BUTTON, LEGEND_BUTTON].forEach(
        btn => btn.addEventListener('click', () => beginQuiz(btn.id))
    );

    [PHANTOM_MENACE, ATTACK_CLONES, REVENGE_SITH, NEW_HOPE, EMPIRE, ROTJ].forEach(
        btn => btn.addEventListener('click', () => movieSelect(btn.id))
    );

    PLAY_BUTTON.addEventListener('click', () => {
     try {
        toAskAudio[currentIndex].play();
     } catch (error) {
         console.error(error.stack);
         console.error(error);
     }
    });

    AGAIN_BUTTON.addEventListener('click', () => {
        stopAll();
        beginQuiz(selectedDifficulty);
    });
    MAIN_BUTTON.addEventListener('click', () => {
        stopAll();
        router(MAIN);
    });

    VOLUME_SLIDER.addEventListener("input", () => setVolume());

    if (navigator.userAgent.match(/iPad/i) || navigator.userAgent.match(/iPhone/i)) {
        // ios doesn't let you adjust volume
        document.getElementById("volume-div").style.display = "none";
    }
}

document.addEventListener("DOMContentLoaded", main);

window.addEventListener("popstate", (event) => {
    try {
        stopAll();
    } catch (err) {
        // lol
    }
    router(MAIN);
})