require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'ca.crt' do
    let(:template) {job.template('config/certs/ca.crt')}

    context 'when configured with a ca_cert' do
      let(:manifest_properties) do
        {
            "tls" => {
                "ca_cert" => "some-ca-cert"
            },
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).to include("some-ca-cert")
      end
    end
  end
end
