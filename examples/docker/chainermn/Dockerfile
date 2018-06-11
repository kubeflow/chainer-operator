FROM chainer/chainer:v4.1.0-python3

ARG OPENMPI_VERSION="2.1.3"
ARG CHAINER_MN_VERSION="1.3.0"

# Install basic dependencies and locales
RUN apt-get update && apt-get install -yq --no-install-recommends \
      locales wget sudo ca-certificates ssh build-essential && \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/* && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen

# Install OpenMPI with cuda
RUN cd /tmp && \
  wget -q https://www.open-mpi.org/software/ompi/v${OPENMPI_VERSION%\.*}/downloads/openmpi-$OPENMPI_VERSION.tar.bz2 && \
  tar -xjf openmpi-$OPENMPI_VERSION.tar.bz2 && \
  cd /tmp/openmpi-$OPENMPI_VERSION && \
  ./configure --prefix=/usr --with-cuda && make -j2 && make install && rm -r /tmp/openmpi-$OPENMPI_VERSION* && \
  ompi_info --parsable --all | grep -q "mpi_built_with_cuda_support:value:true"

# Install ChainerMN
RUN pip3 install chainermn==$CHAINER_MN_VERSION

# Download train_mnist.py example of ChainerMN and download mnist data
# In practice, you would download your codes here.
RUN mkdir -p /chainermn-examples/mnist && \
  cd /chainermn-examples/mnist && \
  wget https://raw.githubusercontent.com/chainer/chainermn/v${CHAINER_MN_VERSION}/examples/mnist/train_mnist.py -O /train_mnist.py && \
  python3 -c "import chainer; chainer.datasets.get_mnist()"
