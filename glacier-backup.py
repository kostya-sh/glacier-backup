import time
import sys
import os
import os.path
import logging
from hashlib import md5

from boto.glacier.layer2 import Layer2

DEBUG = 0
BOTO_DEBUG = 1
CFG_DIR = ".glacier-backup"
USER_CFG_FILE = os.path.join(os.path.expanduser('~'), ".glacier-backup")

def parse_cfg_file(f):
    if not os.path.exists(f):
        return {}
    return dict([(k.strip(), v.strip()) for (k, v) in
        [x.strip().split('=', 2) for x in open(f, 'r') if not x.startswith('#') and not len(x.strip()) == 0]])

def calc_digest(f):
    m = md5()
    m.update(open(f, 'r').read()) # TODO: read in chunks
    return m.hexdigest()

class Config:
     def __init__(self, dir_cfg_file):
        dir_cfg = parse_cfg_file(dir_cfg_file)
        user_cfg = parse_cfg_file(USER_CFG_FILE)

        def get_value(key, default=None):
            return dir_cfg.get(key, user_cfg.get(key, default))

        self.vault = dir_cfg.get("vault")
        self.aws_access_key_id = get_value("aws_access_key_id")
        self.aws_secret_access_key = get_value("aws_secret_access_key")
        self.proxy = get_value("proxy")
        self.proxy_port = get_value("proxy_port")
        if self.proxy_port is not None:
            self.proxy_port = int(self.proxy_port)
        self.region = get_value("region", "us-east-1")
        self.dbfile_size = int(get_value("dbfile_size", "20"))

        logging.debug("DIRECTORY: %s", dir_cfg_file)
        logging.debug("USER CONFIG: %s", user_cfg)
        logging.debug("DIRECTORY CONFIG: %s", dir_cfg)

class File:
    def __init__(self, file_path, file_name, archive_id=None, modified_time=None, backup_time=None, digest=None):
        self.file_name = file_name
        self.file_path = file_path
        self.archive_id = archive_id
        self.modified_time = modified_time
        self.backup_time = backup_time
        self.digest = digest

    @staticmethod
    def parse(line, file_path):
        parts = line.split('\t')
        return File(file_path=file_path,
            file_name=parts[0],
            archive_id=parts[1],
            modified_time=float(parts[2]),
            backup_time=float(parts[3]),
            digest=parts[4])

    def write(self, f):
        f.write(str(self))
        f.write('\n')

    def __str__(self):
        return "%s\t%s\t%f\t%f\t%s" % (self.file_name, self.archive_id, self.modified_time, self.backup_time, self.digest)

def read_uploaded_files(d):
    res = {}
    dbfiles = [fn for fn in os.listdir(d) if fn.endswith(".db")]
    dbfiles.sort() # sort to make sure that latest records take precedense
    for fname in dbfiles:
        file_path = os.path.join(d, fname)
        for line in open(file_path, 'r'):
            line = line.strip()
            if len(line) == 0:
                continue
            try:
                f = File.parse(line, file_path)
                res[f.file_name] = f
            except Exception, e:
                logging.warn("Failed to parse line '%s' from %s: %s", line, fname, e)
    return res

def create_dbfile(d, cnt=0):
    if cnt > 99:
        raise Exception("Unable to create dbfile in %s" % d)
    fname = "%s.%02d.db" % (time.strftime("%Y%m%d_%H%M%S", time.gmtime()), cnt)
    f = os.path.join(d, fname)
    if os.path.exists(f):
        return create_dbfile(d, cnt + 1)
    return open(f, 'w')

def extract_filename(d, fn):
    # TODO: remove multiple '\'
    d = d.replace('\\', '/')
    if not d.endswith('/'):
        d += '/'
    fn = fn.replace('\\', '/')
    return fn.replace(d, "")

def backup(d):
    # parse and validate config
    cfg_dir = os.path.join(d, CFG_DIR)
    cfg = Config(os.path.join(cfg_dir, "config"))
    if cfg.vault is None:
       raise Exception("no vault")
    if cfg.aws_access_key_id is None or cfg.aws_secret_access_key is None:
       raise Exception("no credentials")

    # read list of already uploaded files
    uploaded_files = read_uploaded_files(cfg_dir)

    # find files that need to be uploaded or updated in db files
    files_to_upload = []
    files_to_update = []
    for root, dirs, files in os.walk(d):
        for file_path in [os.path.join(root, name) for name in files]:
            st = os.stat(file_path)
            file_name = extract_filename(d, file_path)
            f = File(file_path, file_name, modified_time=st.st_mtime)
            existing_file = uploaded_files.get(file_name)
            if existing_file is None:
                logging.info("new file: %s (modified_time = %f)", file_name, st.st_mtime)
                files_to_upload.append(f)
            else:
                time_modified = abs(st.st_mtime - existing_file.modified_time) > 1e-6
                content_modified = None
                if time_modified:
                    logging.debug("%s: time changed %f -> %f", file_name, existing_file.modified_time, st.st_mtime)
                    digest = calc_digest(file_path)
                    content_modified = existing_file.digest != digest
                    if content_modified:
                        logging.debug("%s: content changed", file_name)
                if time_modified and content_modified:
                    logging.info("modified file: %s", file_name)
                    f.digest = digest
                    files_to_upload.append(f)
                elif time_modified:
                    logging.info("time updated, but content is the same: %s", file_name)
                    existing_file.modified_time = st.st_mtime
                    files_to_update.append(existing_file)
                else:
                    logging.debug("unmodified file: %s", file_name)
        if CFG_DIR in dirs:
            dirs.remove(CFG_DIR) # do not visit .glacier-backup directory

    if len(files_to_update) != 0:
        dbfile = create_dbfile(cfg_dir) 
        for f in files_to_update:
            f.write(dbfile)
        dbfile.close()
        logging.info("updated times for %d files", len(files_to_update))

    if len(files_to_upload) == 0:
        logging.info("no files to upload")
        return

    # connect to Glacier
    glacier = Layer2(aws_access_key_id=cfg.aws_access_key_id, aws_secret_access_key=cfg.aws_secret_access_key,
        proxy=cfg.proxy, proxy_port=cfg.proxy_port,
        debug=BOTO_DEBUG,
        region_name=cfg.region)
    vault = glacier.get_vault(cfg.vault)

    # upload files
    dbfile = None
    for (i, f) in enumerate(files_to_upload):
        if i % cfg.dbfile_size == 0:
            if dbfile is not None:
                dbfile.close()
            dbfile = create_dbfile(cfg_dir)
            logging.debug("using db file %s", dbfile)
        try:
            archive_id = vault.upload_archive(f.file_path, description=f.file_name)
        except Exception, e:
            logging.warning("failed to upload %s: %s", file_name, e)
            continue

        if f.digest is None:
            f.digest = calc_digest(f.file_path)
        f.archive_id = archive_id
        f.backup_time = time.time()
        f.write(dbfile)
        logging.info("[%d/%d] %s uploaded", i+1, len(files_to_upload), f.file_name)

if __name__ == "__main__":
    level = logging.INFO
    if DEBUG > 0:
        level = logging.DEBUG
    logging.basicConfig(level=level, format='%(asctime)s %(message)s')

    d = sys.argv[1]
    backup(d)
