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

export async function getClipUrl(url) {
    console.log("url!");
    const response = await fetch(`https://cdn.jayd.ml/clips/${url}.enc`);

    const arrayBuf = await response.arrayBuffer();

    const clipBytes = new Uint8Array(arrayBuf);
    const aesCtr = new aesjs.ModeOfOperation.ctr(MEDIA_KEY, MEDIA_IV);
    const decrypted = aesCtr.decrypt(clipBytes);

    const decryptedBlob = new Blob([decrypted], {type: "audio/mpeg"});

    const newUrl = window.URL.createObjectURL(decryptedBlob);
    console.log(newUrl);
    return newUrl;
}