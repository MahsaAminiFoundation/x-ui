import logging
import boto3
from botocore.exceptions import ClientError
import os


def upload_file(file_name, bucket, object_name=None):
    """Upload a file to an S3 bucket

    :param file_name: File to upload
    :param bucket: Bucket to upload to
    :param object_name: S3 object name. If not specified then file_name is used
    :return: True if file was uploaded, else False
    """

    # If S3 object_name was not specified, use file_name
    if object_name is None:
        object_name = os.path.basename(file_name)

    # Upload the file
    s3_client = boto3.client('s3')
    s3 = boto3.resource('s3')
    try:
        response = s3_client.upload_file(file_name, bucket, object_name)
        object_acl = s3.ObjectAcl(bucket, object_name)
        response = object_acl.put(ACL='public-read')
        # response = object_acl.put(ACL='public-read')
    except ClientError as e:
        logging.error(e)
        return False
    return True
    
    
# The config will be uploaded to: https://nofiltervpn.s3.eu-central-1.amazonaws.com/mahsa_amini.vpn.config    
# https://nofiltervpn.s3.amazonaws.com/mahsa_amini.vpn.config    
upload_file("mahsa_amini.vpn.config", "nofiltervpn")
