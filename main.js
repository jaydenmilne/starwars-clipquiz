import {getManifest, getClipUrl, getFirstClip, getNextClip, INCORRECT_GUESS, submitHighscore, getHighscores} from './clips.js';

// main divs
let MAIN_PAGE;
let QUIZ_PAGE;
let RESULTS_PAGE;
let LEADERBOARD_PAGE;

// main screen
let EASY_BUTTON;
let MEDIUM_BUTTON;
let HARD_BUTTON; 
let LEGEND_BUTTON;
let BUTTON_DIV;
let LEADERBOARD_BUTTON_MAIN;

let LOADING_SPINNER;

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
let RESULTS_SCORE_HEADING;
let RESULTS_CORRECT_CHOICE;
let RESULTS_INPUT;
let RESULTS_SUBMIT_BTN;
let RESULTS_USERNAME_DIV;
let RESULTS_CHECK_SPAN;
let RESULTS_VIEW_LEADERBOARD_BUTTON;
let SUBMIT_LEADERBOARD_DIV;

let DURATION;
let AGAIN_BUTTON;
let MAIN_BUTTON;

// leaderboard

let LEADERBOARD_ALL_TIME;
let LEADERBOARD_THIS_WEEK;
let LEADERBOARD_TODAY;
let LEADERBOARD_BACK;
let LEADERBOARD_EASY;
let LEADERBOARD_MEDIUM;
let LEADERBOARD_HARD;
let LEADERBOARD_LEGEND;

const MAIN = "main", QUIZ = "quiz", RESULTS = "results", LEADERBOARD = "leaderboard";
let currentState = MAIN;

const NUM_QUESTIONS = 10;

let selections = new Array();

let toAsk;
let toAskAudio;
let score = 0;
let selectedDifficulty;
let startTime;

let highscores;

// Constants
const idToDisplay = new Map([
    ["phantom-menace", "Episode I"],
    ["attack-clones", "Episode II"],
    ["revenge-sith", "Episode III"],
    ["new-hope", "Episode IV"],
    ["empire", "Episode V"],
    ["rotj", "Episode VI"]
])

function router(newState, pushState=true) {
    currentState = newState;

    if (pushState) {
        window.history.pushState({}, '', `#${newState}`);
    }

    QUIZ_PAGE.style.display = "none";
    RESULTS_PAGE.style.display = "none";
    MAIN_PAGE.style.display = "none";
    LEADERBOARD_PAGE.style.display = "none";
    switch (newState) {
        case MAIN:
            BUTTON_DIV.style.display = "block";
            LOADING_SPINNER.style.display = "none";
            MAIN_PAGE.style.display = "flex";
            break;
        case QUIZ:
            QUIZ_PAGE.style.display = "flex";
            break;
        case RESULTS:
            RESULTS_PAGE.style.display = "flex";
            RESULTS_USERNAME_DIV.style.display = "flex";
            RESULTS_CHECK_SPAN.style.display = "none";
            SUBMIT_LEADERBOARD_DIV.style.display = "block";
            break;
        case LEADERBOARD:
            LEADERBOARD_PAGE.style.display = "flex";
            enterLeaderboard();
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
    const selectedVol = VOLUME_SLIDER.value;
    const vol = Math.exp(9.210 * selectedVol) / 10000;
    console.log(vol);
    toAskAudio.volume = vol;
}

async function beginQuiz(difficultyBtnName) {

    // hide the buttons, display loading
    BUTTON_DIV.style.display = "none";
    LOADING_SPINNER.style.display = "inherit";
    score = 0;
    selectedDifficulty = difficultyBtnName.split('-')[0];

    let clip = await getFirstClip(selectedDifficulty);

    toAskAudio = new Audio(clip);
    toAskAudio.play();
    startTime = new Date()
    router(QUIZ);
}

function updateProgress() {
    PROGRESS.innerText = `Score: ${(score).toString()}`;
}

async function movieSelect(movie) {
    toAskAudio.pause();

    selections.push(movie);

    // see if we did it right
    let nextClip, err;
    try {
        [nextClip, err] = await getNextClip(movie);
    } catch (error) {
        // we didn't 
        alert("OOPS " + error.toString());
        return;
    }

    if (err) {
        // something went wrong
        if (err == INCORRECT_GUESS) {
            // that's all folks!
            showResults(nextClip);
        } else {
            // actual error
            alert(`Error getting clip: ${err}`);
        }
        return;
    }

    score += 1;
    toAskAudio = new Audio(nextClip);
    toAskAudio.play();

    updateProgress();
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

function showResults(correct) {
    stopPlayback();
    
    const quips = [ 
        "WRONG", 
        "Take a seat, young Skywalker", 
        "Oof", 
        "Nope 😭", 
        "¯\\_(ツ)_/¯", 
        "That's no moon...", 
        "You should've had a bad feeling about that", 
        "Clouded, your future is"
    ];

    const [quip] = getRandom(quips, 1);

    RESULTS_HEADING.innerText = quip;
    RESULTS_SCORE_HEADING.innerText = `You got ${score} correct on ${selectedDifficulty}`;
    RESULTS_CORRECT_CHOICE.innerText = `${idToDisplay.get(correct)} was the correct choice`;

    const timeTaken = new Date() - startTime;

    DURATION.innerText = formatDuration(timeTaken / 1000);
    router(RESULTS);

    if (score == 0) {
        // hide the submit div
        SUBMIT_LEADERBOARD_DIV.style.display = "none";
    }
}

function stopPlayback() {
    if (toAskAudio) {
        toAskAudio.pause();
        toAskAudio.currentTime = 0;
    }    
}

function switchLeaderDifficulty(selectedDifficultyName) {
    selectedDifficulty = selectedDifficultyName;

    document.getElementById("leaderboard-difficulty").innerText = `Difficulty: ${selectedDifficultyName}`;

    function renderHighscores(scoreList, listElem) {
        clearChildren(listElem);

        for(let i=0; i<scoreList.length; ++i) {
            const elem = document.createElement('li');
            elem.innerText = `${scoreList[i]["name"]} — ${scoreList[i]["score"]}`;
            listElem.appendChild(elem);
        }
    };

    const diffScores = highscores[selectedDifficulty];

    renderHighscores(diffScores["allTime"], LEADERBOARD_ALL_TIME);
    renderHighscores(diffScores["week"], LEADERBOARD_THIS_WEEK);
    renderHighscores(diffScores["today"], LEADERBOARD_TODAY);
}

function clearChildren(elem) {
    while (elem.firstChild) {
        elem.removeChild(elem.firstChild);
    }
}

async function enterLeaderboard() {
    const [scores, err] = await getHighscores();
    if (err) {
        alert(err);
        return;
    }
    highscores = scores["highscores"]

    if (!selectedDifficulty) {
        selectedDifficulty = "legend";
    }

    switchLeaderDifficulty(selectedDifficulty);

}

function main() {
    EASY_BUTTON = document.getElementById("easy-button");
    MEDIUM_BUTTON = document.getElementById("medium-button");
    HARD_BUTTON = document.getElementById("hard-button");
    LEGEND_BUTTON = document.getElementById("legend-button");
    LEADERBOARD_BUTTON_MAIN = document.getElementById("main-view-leaderboard");

    BUTTON_DIV = document.getElementById("start-div");
    LOADING_SPINNER = document.getElementById("loading-spinner");

    MAIN_PAGE = document.getElementById("main");
    QUIZ_PAGE = document.getElementById("quiz");
    RESULTS_PAGE = document.getElementById("results");
    LEADERBOARD_PAGE = document.getElementById("leaderboard");

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
    DURATION = document.getElementById("time-div");
    AGAIN_BUTTON = document.getElementById("again");
    MAIN_BUTTON = document.getElementById("change-difficulty");
    RESULTS_SCORE_HEADING = document.getElementById("results-score-heading");
    RESULTS_CORRECT_CHOICE = document.getElementById("results-correct");
    RESULTS_INPUT = document.getElementById("results-name");
    RESULTS_SUBMIT_BTN = document.getElementById("submit-username-btn");
    RESULTS_USERNAME_DIV = document.getElementById("username-div");
    RESULTS_CHECK_SPAN = document.getElementById("results-check-span");
    RESULTS_VIEW_LEADERBOARD_BUTTON = document.getElementById("results-view-leaderboard");
    SUBMIT_LEADERBOARD_DIV = document.getElementById("submit-leaderboard");

    // leaderboard

    LEADERBOARD_ALL_TIME = document.getElementById("leaderboard-all-time");
    LEADERBOARD_THIS_WEEK = document.getElementById("leaderboard-this-week");
    LEADERBOARD_TODAY = document.getElementById("leaderboard-today");
    LEADERBOARD_BACK = document.getElementById("leaderboard-back");
    LEADERBOARD_EASY = document.getElementById("easy-button-leaderboard");
    LEADERBOARD_MEDIUM = document.getElementById("medium-button-leaderboard");
    LEADERBOARD_HARD = document.getElementById("hard-button-leaderboard");
    LEADERBOARD_LEGEND = document.getElementById("legend-button-leaderboard");

    // add listeners

    // main page

    [EASY_BUTTON, MEDIUM_BUTTON, HARD_BUTTON, LEGEND_BUTTON].forEach(
        btn => btn.addEventListener('click', () => beginQuiz(btn.id))
    );

    [PHANTOM_MENACE, ATTACK_CLONES, REVENGE_SITH, NEW_HOPE, EMPIRE, ROTJ].forEach(
        btn => btn.addEventListener('click', () => movieSelect(btn.id))
    );

    [RESULTS_VIEW_LEADERBOARD_BUTTON, LEADERBOARD_BUTTON_MAIN].forEach(
        btn => btn.addEventListener('click', () => router(LEADERBOARD))
    )
    // quiz

    VOLUME_SLIDER.addEventListener("input", () => setVolume());

    if (navigator.userAgent.match(/iPad/i) || navigator.userAgent.match(/iPhone/i)) {
        // ios doesn't let you adjust volume
        document.getElementById("volume-div").style.display = "none";
    }

    PLAY_BUTTON.addEventListener('click', () => {
     try {
        toAskAudio.play();
     } catch (error) {
         console.error(error.stack);
         console.error(error);
     }
    });

    // results
    AGAIN_BUTTON.addEventListener('click', () => {
        stopPlayback();
        beginQuiz(selectedDifficulty);
    });
    MAIN_BUTTON.addEventListener('click', () => {
        stopPlayback();
        router(MAIN);
    });


    // leaderboard
    [LEADERBOARD_EASY, LEADERBOARD_MEDIUM, LEADERBOARD_HARD, LEADERBOARD_LEGEND].forEach( 
        (btn) => btn.addEventListener("click", () => switchLeaderDifficulty(btn.id.split('-')[0]))
    );

    LEADERBOARD_BACK.addEventListener("click", () => {
        router(MAIN);
    })

    RESULTS_INPUT.addEventListener('input', () => {
        const len = (new TextEncoder().encode(RESULTS_INPUT.value)).length;
        if (len > 20) {
            RESULTS_SUBMIT_BTN.innerText = "Too long!";
            RESULTS_SUBMIT_BTN.disabled = true;
        } else {
            RESULTS_SUBMIT_BTN.innerText = "Submit";
            RESULTS_SUBMIT_BTN.disabled = false;
        }
    })

    RESULTS_SUBMIT_BTN.addEventListener('click', async () => {
        if (RESULTS_INPUT.value.length == 0) {
            alert("Please put in a name to submit to the leaderboard");
            return;
        }

        const err = await submitHighscore(RESULTS_INPUT.value);
        if (err) {
            alert(`Failed to submit highscore!\n\nError: '${err.trim()}'`);
            return;
        }

        RESULTS_USERNAME_DIV.style.display = "none";
        RESULTS_CHECK_SPAN.style.display = "inherit";
    })

    router(getPageFromHash(), true);
}

document.addEventListener("DOMContentLoaded", main);

function getPageFromHash() {
    let page = window.location.hash.replace("#", "");
    if (page == QUIZ || page == RESULTS || page == "") {
        page = MAIN;
    }
    return page;
}

window.addEventListener("popstate", (event) => {
    try {
        stopPlayback();
    } catch (err) {
        // lol
    }
    router(getPageFromHash(), false);
})