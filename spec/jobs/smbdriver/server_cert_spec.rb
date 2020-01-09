require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'server.crt' do
    let(:template) {job.template('config/certs/server.crt')}

    context 'when configured with a server cert' do
      let(:manifest_properties) do
        {
            "tls" => {
                "server_cert" => "some-server-cert"
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("some-server-cert")
      end
    end
  end
end
