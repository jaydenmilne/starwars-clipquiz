const cache = new Map();

function objectify(file, episode) {
    return {
        episode,
        file
    }
}

const MANIFEST_KEY = aesjs.utils.hex.toBytes("38d028fc7cd802503290a247f671c92400e0f4630f7c850642466dd2f4598805");
const MANIFEST_IV = aesjs.utils.hex.toBytes("d17a4a54ac9bd69f1bcc7bce1223a3df");

const MEDIA_KEY = aesjs.utils.hex.toBytes("f4129dbc91d36973ac24f57a1682d3b7327895ce71429f33ff2b153207d55cc6");
const MEDIA_IV = aesjs.utils.hex.toBytes("bde166cd43d5ecb0fb7930f92e54bac8");

let backendUrl = location.hostname.includes("localhost") || location.hostname.includes("192") ? `http://192.168.1.29:3123/clipquiz/v1` : `https://apistarwars.jayd.ml/clipquiz/v1`;

export async function getManifest(difficulty) {
    let manifestBytes;
    if (cache.has(difficulty)) {
        console.log(`Returning ${difficulty} from cache...`);
        manifestBytes = cache.get(difficulty);
    } else {
        let resp = await fetch(`manifest/${difficulty}.json.enc`);

        const manifestEncBuffer = await resp.arrayBuffer();
        manifestBytes = new Uint8Array(manifestEncBuffer);
        // store the encrypted bytes
        cache.set(difficulty, manifestBytes);
    }

    const aesCtr = new aesjs.ModeOfOperation.ctr(MANIFEST_KEY, MANIFEST_IV);
    const decrypted = aesCtr.decrypt(manifestBytes);

    const manifest = JSON.parse(aesjs.utils.utf8.fromBytes(decrypted));

    let all = [
        ...manifest["phantom-menace"].map((elem) => objectify(elem, "phantom-menace")),
        ...manifest["attack-clones"].map((elem) => objectify(elem, "attack-clones")),
        ...manifest["revenge-sith"].map((elem) => objectify(elem, "revenge-sith")),
        ...manifest["new-hope"].map((elem) => objectify(elem, "new-hope")),
        ...manifest["empire"].map((elem) => objectify(elem, "empire")),
        ...manifest["rotj"].map((elem) => objectify(elem, "rotj"))
    ]
    return all;
}

function encryptedBody2Url(arrayBuf) {
    const clipBytes = new Uint8Array(arrayBuf);
    const aesCtr = new aesjs.ModeOfOperation.ctr(MEDIA_KEY, MEDIA_IV);
    const decrypted = aesCtr.decrypt(clipBytes);

    const decryptedBlob = new Blob([decrypted], {type: "audio/mpeg"});

    const newUrl = window.URL.createObjectURL(decryptedBlob);
    return newUrl;
}

export async function getClipUrl(filename) {
    let url;
    if (location.hostname.includes("localhost")) {
        url = `http://localhost:8001/clips/${filename}.enc`;
    } else {
        url = `https://yodaspincdn.jayd.ml/file/yodaspincdn/clips/${filename}.enc`;
    }

    const response = await fetch(url);
    const arrayBuf = await response.arrayBuffer();

    return encryptedBody2Url(arrayBuf);
}

let token;

export async function getFirstClip(difficulty) {
    const params = new URLSearchParams();
    params.set("difficulty", difficulty);
    const response = await fetch(`${backendUrl}/clip?${params.toString()}`, {
        method: 'POST',
        headers: {
            "Auth-Token": ""
        }});

    if (response.status != 200) {
        console.error(`got bad status ${response.status} from backend`);
        console.error(response);
        return;
    }

    const arrayBuf = await response.arrayBuffer();

    // save header
    token = response.headers.get('Auth-Token');
    if (!token) {
        alert("Missing auth header!");
        return
    }

    return encryptedBody2Url(arrayBuf);
}

export const INCORRECT_GUESS = "¯\\_(ツ)_/¯";

export async function getNextClip(guess) {
    const params = new URLSearchParams();
    params.set("guess", guess)
    const response = await fetch(`${backendUrl}/clip?${params.toString()}`, {
        method: 'POST',
        headers: {
            'Auth-Token': token
        }
    });

    if (response.status == 404) {
        // get the body
        const correct = await response.text();
        return [correct, INCORRECT_GUESS];
    } else if (response.status != 200) {
        console.error(`got bad status ${response.status} from backend`);
        console.error(response);
        return [null, `Got beckend error ${response.status}`];
    }

    // save header
    token = response.headers.get('Auth-Token');
    const arrayBuf = await response.arrayBuffer();
    return [encryptedBody2Url(arrayBuf), null];
}

export async function getHighscores() {
    try {
        const response = await fetch(`${backendUrl}/highscore`);
        if (response.status != 200) {
            return [null, `Got ${response.status} from backend!`];
        }

        return [await response.json(), null];
    } catch (e) {
        return [null, `unhandled exception fetching high scores: ${error.toString()}`];
    }
}

export async function submitHighscore(name) {
    try {
        const params = new URLSearchParams();
        params.set("name", name);
        const response = await fetch(`${backendUrl}/highscore?${params.toString()}`, {
            method: 'POST',
            headers: {
                'Auth-Token': token
            }
        });
    
        if (response.status != 201) {
            console.error(`got bad status ${response.status} from backend`);
            console.error(response);
    
            return await response.text();
        }
    
        return null;
    } catch (error) {
        return `unhandled exception: ${error.toString()}`;
    }

}