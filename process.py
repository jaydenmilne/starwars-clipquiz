import sys
import glob
import os
import shutil
from shutil import copyfile
import uuid
import json
from pathlib import Path


for time in [10, 5, 2, 1]:
    print("RUNNING FOR ", time)

    processed_dir = os.path.join(f'temp', str(time))
    if os.path.exists(processed_dir):
        shutil.rmtree(processed_dir)
    Path(processed_dir).mkdir(parents=True, exist_ok=True)

    inFile = sys.argv[1]
    name = inFile.split('.')[0]
    os.system(f'ffmpeg -i {inFile} -c copy -map 0 -segment_time {time} -f segment {processed_dir}\{name}_{time}_%06d.opus')

    files = glob.glob(processed_dir + '\*')
    uuid_dir = os.path.join(f'out_files', f'{name}', f'{time}')

    Path(uuid_dir).mkdir(parents=True, exist_ok=True)

    written = []

    for file in files:
        filename = str(uuid.uuid4()) + '.opus'
        newfile = os.path.join(uuid_dir, filename)
        copyfile(file, newfile)
        written.append(filename)

    with open(os.path.join(uuid_dir, f'manifest_{time}_{name}.json'), 'w')  as f:
        json.dump(written, f)

